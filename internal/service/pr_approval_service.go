package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"rms-backend/internal/repository"
)

var (
	ErrApprovalSequenceViolation = errors.New("approval level sebelumnya belum disetujui")
	ErrApprovalNotOwnedByActor = errors.New("baris approval ini bukan milik admin yang login")
	ErrApprovalLevelAlreadyExists = errors.New("level approval ini sudah dibuat untuk round yang sama")
	ErrPRNotEligibleForSubmission = errors.New("pr harus berstatus draft atau revision sebelum dapat disubmit")
	ErrInvalidApprovalDecision = errors.New("keputusan approval harus approved atau rejected")
	ErrApprovalNoApprovers = errors.New("minimal satu approver wajib ditentukan")	
	ErrPRNotOwnedByActor = errors.New("purchase request ini bukan milik admin yang login")
	ErrRevisionNotAllowedForLevel = errors.New("approver level ini tidak berwenang meminta revisi (hanya level 1 dan 2)")
	ErrPRNotSubmittedForDecision = errors.New("purchase request ini sudah tidak berstatus submitted, keputusan approval tidak dapat diproses lagi")
	ErrApproverSignatureRequired = errors.New("approver belum memiliki tanda tangan terdaftar, silakan unggah tanda tangan terlebih dahulu sebelum dapat melakukan approval")
	ErrCostControlAttachmentRequired = errors.New("purchase request dengan nominal di atas Rp 50.000.000 wajib melampirkan dokumen cost control sebelum dapat disubmit")
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
	paymentRepo   *repository.PRPaymentRepo // BARU
}

func NewPRApprovalService(
	approvalRepo *repository.PRApprovalRepo,
	prRepo *repository.PRRepo,
	prService *PRService,
	signatureRepo *repository.AdminSignatureRepo,
	documentRepo *repository.PRDocumentRepo,
	paymentRepo *repository.PRPaymentRepo, // BARU
) *PRApprovalService {
	return &PRApprovalService{
		approvalRepo: approvalRepo, prRepo: prRepo, prService: prService,
		signatureRepo: signatureRepo, documentRepo: documentRepo,
		paymentRepo: paymentRepo,
	}
}

func (s *PRApprovalService) SubmitForApproval(ctx context.Context, prID int, actorAdminID string, approvers []ApproverAssignment) error {
	if len(approvers) == 0 {
		return ErrApprovalNoApprovers
	}

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

	// BARU: requester WAJIB punya tanda tangan terdaftar sebelum submit -
	// tanda tangan TERBARU dipakai sebagai "prepared by" pada PR ini.
	// Dipindahkan dari CreatePR karena "prepared by" secara bisnis baru
	// relevan saat PR benar-benar diajukan, bukan saat masih draft.
	sig, err := s.signatureRepo.GetLatestByAdmin(ctx, actorAdminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRequesterSignatureRequired
		}
		return err
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

	for _, a := range approvers {
		_, err := s.approvalRepo.GetByPRLevelRound(ctx, prID, a.Level, newRound)
		if err == nil {
			return ErrApprovalLevelAlreadyExists
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := s.approvalRepo.Create(ctx, prID, a.AdminID, a.Level, a.Type, newRound); err != nil {
			return err
		}
	}

	// Pasang signature SEBELUM ChangeStatus, supaya kalau pemasangan gagal,
	// PR tidak terlanjur berpindah ke status "submitted" tanpa "prepared by".
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

	if approval.RefAdmin != actorAdminID {
		return ErrApprovalNotOwnedByActor
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

	if approval.Level > 1 {
		prev, err := s.approvalRepo.GetByPRLevelRound(ctx, approval.RefPR, approval.Level-1, approval.Round)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrApprovalSequenceViolation
			}
			return err
		}
		if prev.Status != "approved" {
			return ErrApprovalSequenceViolation
		}
	}

	// Persyaratan baru: melakukan APPROVE (bukan reject) mewajibkan approver
	// punya tanda tangan terdaftar. Tanda tangan TERBARU milik admin otomatis
	// dipakai & dicatat sebagai jejak pada baris approval ini.
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

	if err := s.approvalRepo.UpdateStatus(ctx, approvalID, decision, notes, signatureRef); err != nil {
		return err
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
			return nil
		}
	}

	if err := s.prService.ChangeStatus(ctx, approval.RefPR, actorAdminID, "approved", "Seluruh level approval telah disetujui"); err != nil {
		return err
	}
	return s.paymentRepo.ActivateDraftPayments(ctx, approval.RefPR) // BARU
}

func (s *PRApprovalService) RequestRevision(ctx context.Context, approvalID int, actorAdminID, notes string) error {
	approval, err := s.approvalRepo.GetByID(ctx, approvalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if approval.RefAdmin != actorAdminID {
		return ErrApprovalNotOwnedByActor
	}
	if approval.Level >= 3 {
		return ErrRevisionNotAllowedForLevel
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

	if approval.Level > 1 {
		prev, err := s.approvalRepo.GetByPRLevelRound(ctx, approval.RefPR, approval.Level-1, approval.Round)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrApprovalSequenceViolation
			}
			return err
		}
		if prev.Status != "approved" {
			return ErrApprovalSequenceViolation
		}
	}

	if err := s.approvalRepo.UpdateStatus(ctx, approvalID, "revision_requested", notes, nil); err != nil {
		return err
	}

	return s.prService.TransitionToRevision(ctx, approval.RefPR, actorAdminID, notes)
}

var (
	ErrPRNotApprover         = errors.New("hanya approver level 1 pada round approval terbaru PR ini yang dapat mengatur priority")
	ErrPRLevel1NotFound      = errors.New("baris approval level 1 untuk PR ini belum tersedia (PR belum pernah disubmit)")
	ErrPRDraftCannotPriority = errors.New("priority tidak dapat diatur selama PR masih berstatus draft")
)

func (s *PRApprovalService) SetPriority(ctx context.Context, prID int, adminID, priority string) error {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if pr.Status == "draft" {
		return ErrPRDraftCannotPriority
	}

	round, err := s.approvalRepo.GetLatestRound(ctx, prID)
	if err != nil {
		return err
	}

	level1, err := s.approvalRepo.GetByPRLevelRound(ctx, prID, 1, round)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPRLevel1NotFound
	}
	if err != nil {
		return err
	}
	if level1.RefAdmin != adminID {
		return ErrPRNotApprover
	}

	return s.prRepo.UpdatePriority(ctx, prID, priority, adminID)
}