package service

import (
	"context"
	"database/sql"
	"errors"

	"rms-backend/internal/repository"
	"rms-backend/internal/models"
)

var (
	ErrPaymentMakerCheckerViolation = errors.New("admin yang mengonfirmasi pembayaran tidak boleh sama dengan admin yang mencatatnya")
	ErrPaymentAlreadyPaid = errors.New("pembayaran ini sudah dikonfirmasi lunas sebelumnya")
	ErrPaymentAlreadyCancelled = errors.New("pembayaran ini sudah dibatalkan")
)

type PRPaymentService struct {
	paymentRepo *repository.PRPaymentRepo
	commentRepo *repository.PRCommentRepo
	prService   *PRService
}

func NewPRPaymentService(paymentRepo *repository.PRPaymentRepo, commentRepo *repository.PRCommentRepo, prService *PRService) *PRPaymentService {
	return &PRPaymentService{paymentRepo: paymentRepo, commentRepo: commentRepo, prService: prService}
}

func (s *PRPaymentService) CreatePayment(ctx context.Context, prID int, adminInputID string, in repository.PRPaymentInput) (int64, error) {
	if err := s.prService.EnsureApprovedForPayment(ctx, prID); err != nil {
		return 0, err
	}

	in.RefPR = prID
	in.RefAdminInput = adminInputID
	return s.paymentRepo.Create(ctx, in)
}

func (s *PRPaymentService) ConfirmPaid(ctx context.Context, paymentID int, adminPaidID string) error {
	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if payment.Status == "draft" { // BARU
		return ErrPaymentStillDraft
	}
	if payment.Status == "paid" {
		return ErrPaymentAlreadyPaid
	}
	if payment.Status == "cancelled" {
		return ErrPaymentAlreadyCancelled
	}
	if payment.RefAdminInput == adminPaidID {
		return ErrPaymentMakerCheckerViolation
	}

	if err := s.paymentRepo.MarkPaid(ctx, paymentID, adminPaidID); err != nil {
		return err
	}
	return s.prService.EvaluatePaymentCompletion(ctx, payment.RefPR, adminPaidID)
}


func (s *PRPaymentService) CancelPayment(ctx context.Context, paymentID int, actorAdminID, notes string) error {
	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if payment.Status == "paid" {
		return ErrPaymentAlreadyPaid
	}
	if payment.Status == "cancelled" {
		return ErrPaymentAlreadyCancelled
	}

	if err := s.paymentRepo.UpdateStatus(ctx, paymentID, "cancelled"); err != nil {
		return err
	}

	if _, err := s.commentRepo.Insert(ctx, payment.RefPR, actorAdminID, notes, "payment_cancellation"); err != nil {
		return err
	}
	return nil
}

type StagePaymentHistory struct {
	PRID     int
	RfpNo    string
	Status   string
	Payments []models.PRPayment
}

var ErrPaymentStillDraft = errors.New("pembayaran ini masih berstatus draft, PR harus approved terlebih dahulu sebelum dapat dikonfirmasi")

func (s *PRPaymentService) GetPaymentChainHistory(ctx context.Context, prID int) ([]StagePaymentHistory, error) {
	chain, err := s.prService.prRepo.GetReferenceChain(ctx, prID) // lihat catatan di bawah
	if err != nil {
		return nil, err
	}
	out := make([]StagePaymentHistory, 0, len(chain))
	for _, pr := range chain {
		payments, err := s.paymentRepo.ListByPR(ctx, pr.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, StagePaymentHistory{
			PRID: pr.ID, RfpNo: pr.RfpNo, Status: pr.Status, Payments: payments,
		})
	}
	return out, nil
}