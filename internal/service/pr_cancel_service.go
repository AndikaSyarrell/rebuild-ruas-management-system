package service

import (
	"context"
	"database/sql"
	"errors"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
)

var (
	ErrPRCancelRequiresRequest    = errors.New("purchase request sudah disetujui checker, pembatalan harus melalui request pembatalan dengan berita acara")
	ErrPRCancelRequestNotRequired = errors.New("purchase request ini masih dapat dibatalkan langsung, tidak perlu request pembatalan")
	ErrCancelReviewerIsRequester  = errors.New("finance tidak dapat mereview request pembatalan miliknya sendiri")
)

type PRCancelService struct {
	cancelRepo   *repository.PRCancelRepo
	prRepo       *repository.PRRepo
	approvalRepo *repository.PRApprovalRepo
	prService    *PRService
}

func NewPRCancelService(cancelRepo *repository.PRCancelRepo, prRepo *repository.PRRepo,
	approvalRepo *repository.PRApprovalRepo, prService *PRService) *PRCancelService {
	return &PRCancelService{cancelRepo: cancelRepo, prRepo: prRepo, approvalRepo: approvalRepo, prService: prService}
}

// needsRequest: true bila checker (L1) sudah approve pada round berjalan, atau PR sudah approved.
func (s *PRCancelService) needsRequest(ctx context.Context, pr *models.PurchaseRequest) (bool, error) {
	switch pr.Status {
	case "approved":
		return true, nil
	case "submitted":
		round, err := s.approvalRepo.GetLatestRound(ctx, pr.ID)
		if err != nil {
			return false, err
		}
		if round == 0 {
			return false, nil
		}
		l1, err := s.approvalRepo.GetByPRLevelRound(ctx, pr.ID, 1, round)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return l1.Status == "approved", nil
	}
	return false, nil
}

func (s *PRCancelService) loadOwned(ctx context.Context, prID int, adminID string) (*models.PurchaseRequest, error) {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if pr.RefAdmin != adminID {
		return nil, ErrPRCancelNotOwner
	}
	return pr, nil
}

// CancelDirect menggantikan PRHandler.Cancel: batal langsung hanya bila belum lolos checker.
func (s *PRCancelService) CancelDirect(ctx context.Context, prID int, adminID, notes string) error {
	pr, err := s.loadOwned(ctx, prID, adminID)
	if err != nil {
		return err
	}
	switch pr.Status {
	case "draft", "revision", "submitted", "approved":
	default:
		return ErrPRAlreadyFinalized
	}
	need, err := s.needsRequest(ctx, pr)
	if err != nil {
		return err
	}
	if need {
		return ErrPRCancelRequiresRequest
	}
	return s.prService.CancelPR(ctx, prID, adminID, notes)
}

func (s *PRCancelService) CreateRequest(ctx context.Context, prID int, adminID, reason, docName, docPath string) (int64, error) {
	pr, err := s.loadOwned(ctx, prID, adminID)
	if err != nil {
		return 0, err
	}
	if pr.Status == "draft" || pr.Status == "revision" {
		return 0, ErrPRCancelRequestNotRequired
	}
	if pr.Status != "submitted" && pr.Status != "approved" {
		return 0, ErrPRAlreadyFinalized
	}
	need, err := s.needsRequest(ctx, pr)
	if err != nil {
		return 0, err
	}
	if !need {
		return 0, ErrPRCancelRequestNotRequired
	}
	pending, err := s.cancelRepo.HasPendingByPR(ctx, prID)
	if err != nil {
		return 0, err
	}
	if pending {
		return 0, repository.ErrCancelRequestAlreadyPending
	}
	return s.cancelRepo.Insert(ctx, prID, adminID, reason, docName, docPath)
}

func (s *PRCancelService) loadForReview(ctx context.Context, cancelID int, financeID string) (*repository.PRCancelRequestView, error) {
	req, err := s.cancelRepo.GetByID(ctx, cancelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if req.RefAdmin == financeID {
		return nil, ErrCancelReviewerIsRequester
	}
	if req.Status != "pending" {
		return nil, repository.ErrCancelRequestNotPending
	}
	return req, nil
}

func (s *PRCancelService) Approve(ctx context.Context, cancelID int, financeID, notes string) error {
	if _, err := s.loadForReview(ctx, cancelID, financeID); err != nil {
		return err
	}
	return s.cancelRepo.Approve(ctx, cancelID, financeID, notes)
}

func (s *PRCancelService) Reject(ctx context.Context, cancelID int, financeID, notes string) error {
	if _, err := s.loadForReview(ctx, cancelID, financeID); err != nil {
		return err
	}
	return s.cancelRepo.Reject(ctx, cancelID, financeID, notes)
}