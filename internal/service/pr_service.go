package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"rms-backend/internal/repository"
	"rms-backend/internal/models"
)

var (
	ErrPRInvalidStatus            = errors.New("status pr tidak valid")
	ErrPRNotApproved              = errors.New("pr harus berstatus approved sebelum dapat masuk siklus pembayaran")
	ErrRequesterSignatureRequired = errors.New("admin belum memiliki tanda tangan terdaftar, silakan unggah tanda tangan terlebih dahulu sebelum dapat membuat purchase request")
	ErrAdminHasNoDivision         = errors.New("admin belum terhubung ke divisi manapun, tidak dapat membuat nomor RFP")
	ErrPRAlreadyFinalized 		  = errors.New("purchase request sudah berada pada status final (approved/completed/rejected/cancelled) dan tidak dapat dibatalkan")
	ErrPRCancelNotOwner  	 	  = errors.New("hanya requester pemilik purchase request yang dapat membatalkannya")	
	ErrPreviousPRNotFound         = errors.New("purchase request referensi tidak ditemukan")
	ErrPreviousPRNotOwned         = errors.New("purchase request referensi bukan milik admin yang sama")
	ErrPreviousPRCrossResponsible = errors.New("purchase request referensi harus memiliki responsible (CoA) yang sama")
	ErrPreviousPRCircular         = errors.New("referensi purchase request tidak boleh membentuk siklus")
)

var prValidStatuses = map[string]bool{
	"draft":     true,
	"submitted": true,
	"approved":  true,
	"rejected":  true,
	"revision":  true,
	"completed": true,
	"cancelled": true,
}

const maxPRChainDepth = 20 // safety net, bukan batas bisnis

type PRService struct {
	prRepo        *repository.PRRepo
	historyRepo   *repository.PRHistoryRepo
	paymentRepo   *repository.PRPaymentRepo
	signatureRepo *repository.AdminSignatureRepo
	adminRepo     *repository.AdminRepo
	divisionRepo  *repository.DivisionRepo
	counterRepo   *repository.PRCounterRepo
}

func NewPRService(
	prRepo *repository.PRRepo,
	historyRepo *repository.PRHistoryRepo,
	paymentRepo *repository.PRPaymentRepo,
	signatureRepo *repository.AdminSignatureRepo,
	adminRepo *repository.AdminRepo,
	divisionRepo *repository.DivisionRepo,
	counterRepo *repository.PRCounterRepo,
) *PRService {
	return &PRService{
		prRepo: prRepo, historyRepo: historyRepo, paymentRepo: paymentRepo,
		signatureRepo: signatureRepo, adminRepo: adminRepo,
		divisionRepo: divisionRepo, counterRepo: counterRepo,
	}
}

func (s *PRService) GetReferenceChain(ctx context.Context, prID int) ([]models.PurchaseRequest, error) {
	return s.prRepo.GetReferenceChain(ctx, prID)
}

func (s *PRService) generateRfpNo(ctx context.Context, adminID string) (string, error) {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return "", err
	}
	if admin.RefDivision == nil {
		return "", ErrAdminHasNoDivision
	}

	division, err := s.divisionRepo.GetByID(ctx, *admin.RefDivision)
	if err != nil {
		return "", err
	}

	// Counter dibuat per divisi dan tahun agar nomor tidak bergantung pada user
	// serta kembali ke awal ketika memasuki tahun baru.
	year := time.Now().Format("2006")
	counterKey := fmt.Sprintf("%v-%s", *admin.RefDivision, year)
	seq, err := s.counterRepo.NextSequence(ctx, counterKey)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%03d%s", division.Code, seq, year), nil
}

func (s *PRService) CreatePR(ctx context.Context, adminID string, in repository.PRInput) (int64, error) {
	if in.RefPreviousPR != nil {
		if err := s.ValidatePreviousPRReference(ctx, adminID, in.RefResponsible, 0, *in.RefPreviousPR); err != nil {
			return 0, err
		}
	}

	rfpNo, err := s.generateRfpNo(ctx, adminID)
	if err != nil {
		return 0, err
	}
	in.RefAdmin = adminID
	in.RfpNo = rfpNo
	in.SignatureRef = 0 // masih NULL - dipasang saat Submit ("prepared by")
	return s.prRepo.Create(ctx, in)
}

func (s *PRService) ValidatePreviousPRReference(ctx context.Context, actorAdminID string, refResponsible int, currentPRID int, previousPRID int) error {
	if previousPRID == currentPRID {
		return ErrPreviousPRCircular
	}

	prev, err := s.prRepo.GetByID(ctx, previousPRID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPreviousPRNotFound
		}
		return err
	}
	if prev.RefAdmin != actorAdminID {
		return ErrPreviousPRNotOwned
	}
	if prev.RefResponsible != refResponsible {
		return ErrPreviousPRCrossResponsible
	}

	return s.detectCircularReference(ctx, previousPRID, currentPRID)
}

func (s *PRService) detectCircularReference(ctx context.Context, startID, forbiddenID int) error {
	seen := map[int]bool{}
	currentID := startID
	for i := 0; i < maxPRChainDepth; i++ {
		if currentID == forbiddenID {
			return ErrPreviousPRCircular
		}
		if seen[currentID] {
			return ErrPreviousPRCircular // data korup/sudah melingkar sebelumnya
		}
		seen[currentID] = true

		pr, err := s.prRepo.GetByID(ctx, currentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if pr.RefPreviousPR == nil {
			return nil
		}
		currentID = *pr.RefPreviousPR
	}
	return ErrPreviousPRCircular // rantai terlalu panjang, indikasi data korup
}

func (s *PRService) ChangeStatus(ctx context.Context, prID int, adminID, newStatus, notes string) error {
	if !prValidStatuses[newStatus] {
		return ErrPRInvalidStatus
	}

	current, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if err := s.prRepo.UpdateStatus(ctx, prID, newStatus); err != nil {
		return err
	}

	fromStatus := current.Status
	if _, err := s.historyRepo.Insert(ctx, prID, adminID, &fromStatus, newStatus, notes); err != nil {
		return err
	}

	return nil
}

func (s *PRService) TransitionToRevision(ctx context.Context, prID int, adminID, notes string) error {
	return s.ChangeStatus(ctx, prID, adminID, "revision", notes)
}

func (s *PRService) TransitionToRejected(ctx context.Context, prID int, adminID, notes string) error {
	return s.ChangeStatus(ctx, prID, adminID, "rejected", notes)
}

func (s *PRService) EnsureApprovedForPayment(ctx context.Context, prID int) error {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if pr.Status != "approved" {
		return ErrPRNotApproved
	}
	return nil
}

func (s *PRService) EvaluatePaymentCompletion(ctx context.Context, prID int, adminID string) error {
	counts, err := s.paymentRepo.CountByStatus(ctx, prID)
	if err != nil {
		return err
	}

	total := 0
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return nil
	}

	paid, allPaid := counts["paid"]
	if !allPaid || paid != total {
		return nil
	}

	return s.ChangeStatus(ctx, prID, adminID, "completed", "Seluruh pembayaran telah lunas")
}

func (s *PRService) CancelPR(ctx context.Context, prID int, adminID, notes string) error {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if pr.RefAdmin != adminID {
		return ErrPRCancelNotOwner
	}

	switch pr.Status {
	case "draft", "revision", "submitted":
		// boleh dibatalkan
	default:
		return ErrPRAlreadyFinalized
	}

	return s.ChangeStatus(ctx, prID, adminID, "cancelled", notes)
}