package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"rms-backend/internal/repository"
)

var (
	// ErrPONotPrepared              = errors.New("po harus berstatus prepared sebelum dapat di-link ke quotation")
	ErrPONotLinkable              = errors.New("po harus berstatus prepared, progress, atau complete sebelum dapat di-link ke quotation")
	ErrQuotationAlreadyLinked     = errors.New("quotation ini sudah ter-link ke po lain")
	ErrQuotationNotEligible       = errors.New("belum ada purchase request anggota grup quotation ini yang berstatus completed")
	ErrQuotationNotLinkedToThisPO = errors.New("quotation ini tidak ter-link ke po yang dimaksud")
)

type POService struct {
	poRepo        *repository.PORepo
	quotationRepo *repository.PRQuotationRepo
	commentRepo   *repository.PRCommentRepo
}

var linkablePOStatuses = map[string]bool{"prepared": true, "progress": true, "complete": true}

func IsLinkablePOStatus(status string) bool { return linkablePOStatuses[status] }

func NewPOService(poRepo *repository.PORepo, quotationRepo *repository.PRQuotationRepo, commentRepo *repository.PRCommentRepo) *POService {
	return &POService{poRepo: poRepo, quotationRepo: quotationRepo, commentRepo: commentRepo}
}

func (s *POService) ListQuotationCandidates(ctx context.Context, poID, keyword string) ([]repository.QuotationCandidate, error) {
	usedHint := false
	if keyword == "" {
		if po, err := s.poRepo.GetDetail(ctx, poID); err == nil && po.QuotHint != "" {
			keyword, usedHint = po.QuotHint, true
		}
	}
	quotations, err := s.quotationRepo.ListCandidates(ctx, keyword)
	if err != nil {
		return nil, err
	}
	if usedHint && len(quotations) == 0 {
		if quotations, err = s.quotationRepo.ListCandidates(ctx, ""); err != nil {
			return nil, err
		}
	}

	out := make([]repository.QuotationCandidate, 0, len(quotations))
	for _, q := range quotations {
		members, err := s.quotationRepo.MemberPRs(ctx, q.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, repository.QuotationCandidate{Quotation: q, Members: members})
	}
	return out, nil
}

func (s *POService) LinkQuotation(ctx context.Context, poID string, quotationID int, adminID string) error {
	po, err := s.poRepo.GetDetail(ctx, poID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !IsLinkablePOStatus(po.Status) {
		return ErrPONotLinkable
	}

	quotation, err := s.quotationRepo.GetByID(ctx, quotationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if quotation.RefPO != nil {
		return ErrQuotationAlreadyLinked
	}

	hasCompleted, err := s.quotationRepo.HasCompletedMember(ctx, quotationID)
	if err != nil {
		return err
	}
	if !hasCompleted {
		return ErrQuotationNotEligible
	}

	err = s.quotationRepo.LinkToPO(ctx, quotationID, poID, po.OrderNum, adminID)
	if errors.Is(err, repository.ErrQuotationLinkConflict){
		return  ErrQuotationAlreadyLinked
	}
	return err
}

func (s *POService) UnlinkQuotation(ctx context.Context, poID string, quotationID int, adminID, notes string) error {
	quotation, err := s.quotationRepo.GetByID(ctx, quotationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if quotation.RefPO == nil || *quotation.RefPO != poID {
		return ErrQuotationNotLinkedToThisPO
	}

	label := poID
	if po, err := s.poRepo.GetDetail(ctx, poID); err == nil {
		label = po.OrderNum
	}

	note := fmt.Sprintf("Link ke PO %s dicabut manual oleh admin. Alasan: %s", label, notes)

	err = s.quotationRepo.UnlinkWithAudit(ctx, quotationID, poID, adminID, note)
	if errors.Is(err, repository.ErrQuotationLinkConflict){
		return ErrQuotationNotLinkedToThisPO
	}
	return err
}

func (s *POService) Delete(ctx context.Context, poID, adminID string) error {
	label := poID
	po, err := s.poRepo.GetDetail(ctx, poID)
	if errors.Is(err, sql.ErrNoRows){
		return  ErrNotFound
	}
	if err == nil {
		label = po.OrderNum
	}

	note := fmt.Sprintf("PO %s dihapus - link quotation ke PO ini dicabut otomatis.", label)

	return s.poRepo.DeleteWithQuotationCascade(ctx, poID, adminID, note)
}
