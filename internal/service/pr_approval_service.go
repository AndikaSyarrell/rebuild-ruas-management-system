package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"rms-backend/internal/repository"
	"rms-backend/internal/models"
)

var (
	ErrApprovalSequenceViolation = errors.New("approval level sebelumnya belum disetujui")
	ErrApprovalNotOwnedByActor = errors.New("baris approval ini bukan milik admin yang login")
	ErrApprovalLevelAlreadyExists = errors.New("level approval ini sudah dibuat untuk round yang sama")
	ErrPRNotEligibleForSubmission = errors.New("pr harus berstatus draft atau revision sebelum dapat disubmit")
	ErrInvalidApprovalDecision = errors.New("keputusan approval harus approved atau rejected")
	ErrApprovalNoApprovers = errors.New("minimal satu approver wajib ditentukan")	
	ErrPRNotOwnedByActor = errors.New("purchase request ini bukan milik admin yang login")
	ErrRevisionNotAllowedForLevel = errors.New("permintaan revisi hanya dapat dilakukan pada tahap checker")
	ErrPRNotSubmittedForDecision = errors.New("purchase request ini sudah tidak berstatus submitted, keputusan approval tidak dapat diproses lagi")
	ErrApproverSignatureRequired = errors.New("approver belum memiliki tanda tangan terdaftar, silakan unggah tanda tangan terlebih dahulu sebelum dapat melakukan approval")
	ErrCostControlAttachmentRequired = errors.New("purchase request dengan nominal di atas Rp 50.000.000 wajib melampirkan dokumen cost control sebelum dapat disubmit")

	ErrCheckerInvalid           = errors.New("checker yang dipilih tidak aktif atau tidak memiliki akses checker")
	ErrCheckerIsRequester       = errors.New("requester tidak dapat menjadi checker untuk purchase request miliknya sendiri")
	ErrApprovalAlreadyDecided   = errors.New("baris approval ini sudah diputuskan")
	ErrRejectNotAllowedForLevel = errors.New("reject hanya dapat dilakukan pada tahap checker")
	ErrApproverAccessRequired   = errors.New("admin tidak memiliki akses untuk tahap approval ini")
	ErrApproverIsRequester      = errors.New("requester tidak dapat menyetujui purchase request miliknya sendiri")
	ErrSignatureNotOwnedByActor = errors.New("tanda tangan yang dipilih bukan milik admin yang login")
)

const (
	ApprovalTypeChecker  = "checker"
	ApprovalTypeDirector = "director"
	ApprovalTypeFinance  = "finance"
)

const costControlThreshold = 50_000_000
const costControlDocType = "cost_control"

type ApproverAssignment struct {
	Level   int
	Type    string
	AdminID string
}

type PRApprovalService struct {
	approvalRepo  *repository.PRApprovalRepo
	prRepo        *repository.PRRepo
	prService     *PRService
	signatureRepo *repository.AdminSignatureRepo
	documentRepo  *repository.PRDocumentRepo
	paymentRepo   *repository.PRPaymentRepo
	accessRepo    *repository.AccessRepo // BARU - validasi slug checker/director/finance
}

func NewPRApprovalService(
	approvalRepo *repository.PRApprovalRepo,
	prRepo *repository.PRRepo,
	prService *PRService,
	signatureRepo *repository.AdminSignatureRepo,
	documentRepo *repository.PRDocumentRepo,
	paymentRepo *repository.PRPaymentRepo,
	accessRepo *repository.AccessRepo,
) *PRApprovalService {
	return &PRApprovalService{
		approvalRepo: approvalRepo, prRepo: prRepo, prService: prService,
		signatureRepo: signatureRepo, documentRepo: documentRepo,
		paymentRepo: paymentRepo, accessRepo: accessRepo,
	}
}

func (s *PRApprovalService) SubmitForApproval(ctx context.Context, prID int, actorAdminID string, signatureID int, checkerID string) error {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if pr.RefAdmin != actorAdminID {
		return ErrPRNotOwnedByActor
	}
	if pr.Status != "draft" && pr.Status != "revision" {
		return ErrPRNotEligibleForSubmission
	}

	if checkerID == actorAdminID {
		return ErrCheckerIsRequester
	}
	isChecker, err := s.accessRepo.HasAccess(ctx, checkerID, ApprovalTypeChecker) // sekaligus cek admin aktif
	if err != nil {
		return err
	}
	if !isChecker {
		return ErrCheckerInvalid
	}

	sig, err := s.signatureRepo.GetByID(ctx, signatureID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRequesterSignatureRequired
		}
		return err
	}
	if sig.RefAdmin != actorAdminID {
		return ErrSignatureNotOwnedByActor
	}

	if pr.RequestedAmount > costControlThreshold {
		hasAttachment, err := s.documentRepo.ExistsByPRAndType(ctx, prID, costControlDocType)
		if err != nil {
			return err
		}
		if !hasAttachment {
			return ErrCostControlAttachmentRequired
		}
	}

	latestRound, err := s.approvalRepo.GetLatestRound(ctx, prID)
	if err != nil {
		return err
	}
	newRound := latestRound + 1

	seeds := []repository.ApprovalSeed{
		{Level: 1, Type: ApprovalTypeChecker, RefAdmin: &checkerID},
		{Level: 2, Type: ApprovalTypeDirector},
		{Level: 3, Type: ApprovalTypeFinance},
	}
	if err := s.approvalRepo.CreateRound(ctx, prID, newRound, seeds); err != nil {
		return err
	}

	if err := s.prRepo.UpdateSignature(ctx, prID, sig.ID); err != nil {
		return err
	}

	notes := fmt.Sprintf("PR disubmit untuk approval round %d", newRound)
	return s.prService.ChangeStatus(ctx, prID, actorAdminID, "submitted", notes)
}


func (s *PRApprovalService) Decide(ctx context.Context, approvalID int, actorAdminID, decision, notes string) error {
	if decision != "approved" && decision != "rejected" {
		return ErrInvalidApprovalDecision
	}

	approval, err := s.approvalRepo.GetByID(ctx, approvalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	pr, err := s.prRepo.GetByID(ctx, approval.RefPR)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if err := s.authorizeApprover(ctx, approval, pr, actorAdminID); err != nil {
		return err
	}
	if pr.Status != "submitted" {
		return ErrPRNotSubmittedForDecision
	}
	if approval.Status != "pending" {
		return ErrApprovalAlreadyDecided
	}
	if decision == "rejected" && approval.Level != 1 {
		return ErrRejectNotAllowedForLevel
	}
	if err := s.ensurePreviousLevelApproved(ctx, approval); err != nil {
		return err
	}

	var signatureRef *int
	if decision == "approved" {
		sig, err := s.signatureRepo.GetLatestByAdmin(ctx, actorAdminID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrApproverSignatureRequired
			}
			return err
		}
		signatureRef = &sig.ID
	}

	changed, err := s.approvalRepo.Decide(ctx, approvalID, actorAdminID, decision, notes, signatureRef)
	if err != nil {
		return err
	}
	if !changed {
		return ErrApprovalAlreadyDecided
	}

	if decision == "rejected" {
		return s.prService.TransitionToRejected(ctx, approval.RefPR, actorAdminID, notes)
	}

	siblings, err := s.approvalRepo.ListByPRRound(ctx, approval.RefPR, approval.Round)
	if err != nil {
		return err
	}
	for _, sib := range siblings {
		if sib.Level > approval.Level {
			return nil // masih ada level berikutnya
		}
	}

	if err := s.prService.ChangeStatus(ctx, approval.RefPR, actorAdminID, "approved", "Seluruh level approval telah disetujui"); err != nil {
		return err
	}
	return s.paymentRepo.ActivateDraftPayments(ctx, approval.RefPR)
}

// authorizeApprover: L1 = harus checker yang ditugaskan; L2/L3 = siapa pun yang
// memegang slug sesuai approval_type baris tsb (director/finance).
func (s *PRApprovalService) authorizeApprover(ctx context.Context, a *models.PRApproval, pr *models.PurchaseRequest, actor string) error {
	if a.Level == 1 {
		if a.RefAdmin == nil || *a.RefAdmin != actor {
			return ErrApprovalNotOwnedByActor
		}
		return nil
	}
	if pr.RefAdmin == actor {
		return ErrApproverIsRequester
	}
	// toleransi PR lama yang L2/L3-nya sudah ditugaskan ke orang tertentu
	if a.RefAdmin != nil && *a.RefAdmin == actor {
		return nil
	}
	ok, err := s.accessRepo.HasAccess(ctx, actor, a.Type)
	if err != nil {
		return err
	}
	if !ok {
		return ErrApproverAccessRequired
	}
	return nil
}

func (s *PRApprovalService) ensurePreviousLevelApproved(ctx context.Context, a *models.PRApproval) error {
	if a.Level <= 1 {
		return nil
	}
	prev, err := s.approvalRepo.GetByPRLevelRound(ctx, a.RefPR, a.Level-1, a.Round)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrApprovalSequenceViolation
		}
		return err
	}
	if prev.Status != "approved" {
		return ErrApprovalSequenceViolation
	}
	return nil
}

func (s *PRApprovalService) RequestRevision(ctx context.Context, approvalID int, actorAdminID, notes string) error {
	approval, err := s.approvalRepo.GetByID(ctx, approvalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if approval.Level != 1 {
		return ErrRevisionNotAllowedForLevel
	}
	if approval.RefAdmin == nil || *approval.RefAdmin != actorAdminID {
		return ErrApprovalNotOwnedByActor
	}
	if approval.Status != "pending" {
		return ErrApprovalAlreadyDecided
	}

	pr, err := s.prRepo.GetByID(ctx, approval.RefPR)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if pr.Status != "submitted" {
		return ErrPRNotSubmittedForDecision
	}

	changed, err := s.approvalRepo.Decide(ctx, approvalID, actorAdminID, "revision_requested", notes, nil)
	if err != nil {
		return err
	}
	if !changed {
		return ErrApprovalAlreadyDecided
	}
	return s.prService.TransitionToRevision(ctx, approval.RefPR, actorAdminID, notes)
}

type BulkSubmitItem struct {
	PRID      int
	CheckerID string
}

type BulkSubmitResult struct {
	PRID    int    `json:"pr_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *PRApprovalService) BulkSubmitForApproval(ctx context.Context, actorAdminID string, signatureID int, items []BulkSubmitItem) []BulkSubmitResult {
	results := make([]BulkSubmitResult, 0, len(items))
	for _, it := range items {
		if err := s.SubmitForApproval(ctx, it.PRID, actorAdminID, signatureID, it.CheckerID); err != nil {
			results = append(results, BulkSubmitResult{PRID: it.PRID, Success: false, Error: submitErrorMessage(err)})
			continue
		}
		results = append(results, BulkSubmitResult{PRID: it.PRID, Success: true})
	}
	return results
}

func submitErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "Purchase request tidak ditemukan"
	case errors.Is(err, ErrPRNotEligibleForSubmission):
		return "Purchase request harus berstatus draft atau revision sebelum dapat disubmit"
	case errors.Is(err, ErrRequesterSignatureRequired):
		return "Requester belum memiliki tanda tangan terdaftar"
	case errors.Is(err, ErrApprovalLevelAlreadyExists):
		return "Level approval ini sudah dibuat untuk round yang sama"
	case errors.Is(err, ErrCheckerInvalid):
		return "Checker yang dipilih tidak aktif atau tidak memiliki akses checker"
	case errors.Is(err, ErrCheckerIsRequester):
		return "Requester tidak dapat menjadi checker untuk PR miliknya sendiri"
	case errors.Is(err, ErrSignatureNotOwnedByActor):
		return "Tanda tangan yang dipilih bukan milik Anda"
	case errors.Is(err, ErrPRNotOwnedByActor):
		return "Purchase request ini bukan milik admin yang login"
	case errors.Is(err, ErrCostControlAttachmentRequired):
		return "Dokumen cost control wajib diunggah terlebih dahulu (nominal di atas Rp 50.000.000)"
	case errors.Is(err, ErrSignatureNotOwnedByActor): 
		return "Tanda tangan yang dipilih bukan milik Anda"
	default:
		return "Gagal submit purchase request"
	}
}

type BulkDecideItem struct {
	ApprovalID int
	Notes      string
}

type BulkDecideResult struct {
	ApprovalID int    `json:"approval_id"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func (s *PRApprovalService) BulkDecide(ctx context.Context, actorAdminID, decision string, items []BulkDecideItem) []BulkDecideResult {
	results := make([]BulkDecideResult, 0, len(items))
	for _, it := range items {
		err := s.Decide(ctx, it.ApprovalID, actorAdminID, decision, it.Notes)
		if err != nil {
			results = append(results, BulkDecideResult{
				ApprovalID: it.ApprovalID, Success: false, Error: decideErrorMessage(err),
			})
			continue
		}
		results = append(results, BulkDecideResult{ApprovalID: it.ApprovalID, Success: true})
	}
	return results
}

func decideErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "Baris approval tidak ditemukan"
	case errors.Is(err, ErrApprovalNotOwnedByActor):
		return "Baris approval ini bukan milik admin yang login"
	case errors.Is(err, ErrApprovalSequenceViolation):
		return "Approval level sebelumnya belum disetujui"
	case errors.Is(err, ErrPRNotSubmittedForDecision):
		return "Purchase request ini sudah tidak berstatus submitted"
	case errors.Is(err, ErrInvalidApprovalDecision):
		return "Keputusan approval harus approved atau rejected"
	case errors.Is(err, ErrApproverSignatureRequired):
		return "Approver belum memiliki tanda tangan terdaftar"
	default:
		return "Gagal memproses keputusan approval"
	}
}