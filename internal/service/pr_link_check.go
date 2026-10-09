package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
)

const (
	LinkLevelInfo    = "info"
	LinkLevelOK      = "ok"
	LinkLevelWarning = "warning"
	LinkLevelError   = "error"
)

var ErrResponsibleMismatchWithQuotation = errors.New("responsible (CoA) harus sama dengan responsible pada grup quotation ini")

type LinkCheckInput struct {
	QoutNo      string
	PoNo        string
	ExcludePRID int // PR yang sedang diedit; tidak ikut dihitung saat menentukan responsible grup
}

type LinkCheckPO struct {
	ID                 string
	OrderNum           string
	Status             string
	SubClient          string
	LinkedQuotationNos []string
}

type ResponsibleLock struct {
	Locked   bool
	Conflict bool
	Source   string // "quotation" | "po"
	Items    []repository.QuotationResponsible
}

type LinkCheckResult struct {
	Level   string
	Code    string
	Message string

	QuotationNo       string // ter-normalisasi
	QuotationExists   bool
	QuotationID       int
	QuotationLinkedPO *LinkCheckPO
	Members           []repository.QuotationMember

	PO          *LinkCheckPO
	PoNoDisplay string
	Responsible *ResponsibleLock // nil = tidak ada yang bisa dikunci
}

func toLinkCheckPO(po *models.PO, linkedNos []string) *LinkCheckPO {
	sub := ""
	if po.SubClient != nil {
		sub = *po.SubClient
	}
	return &LinkCheckPO{ID: po.ID, OrderNum: po.OrderNum, Status: po.Status, SubClient: sub, LinkedQuotationNos: linkedNos}
}

func (s *PRService) SearchPOOptions(ctx context.Context, keyword string, limit int) ([]repository.POLinkOption, error) {
	return s.poRepo.SearchForLinking(ctx, keyword, limit)
}

// CheckQuotationPOLink memvalidasi hubungan nomor quotation dan nomor PO yang
// diisi di form PR, sekaligus menentukan responsible yang harus dikunci.
func (s *PRService) CheckQuotationPOLink(ctx context.Context, in LinkCheckInput) (*LinkCheckResult, error) {
	norm := repository.NormalizeQuotationNo(in.QoutNo)
	poNo := strings.TrimSpace(in.PoNo)

	res := &LinkCheckResult{
		Level: LinkLevelInfo, Code: "empty", QuotationNo: norm,
		PoNoDisplay: PoNotReleasedLabel,
		Message: "Isi nomor quotation dan/atau nomor PO untuk memeriksa link.",
	}
	if norm == "" && poNo == "" {
		return res, nil
	}

	// 1. Quotation
	if norm != "" {
		q, err := s.quotationRepo.GetByNo(ctx, norm)
		switch {
		case err == nil:
			res.QuotationExists = true
			res.QuotationID = q.ID
			members, err := s.quotationRepo.MemberPRs(ctx, q.ID)
			if err != nil {
				return nil, err
			}
			res.Members = members
			if q.RefPO != nil {
				linked, err := s.poRepo.GetDetail(ctx, *q.RefPO)
				switch {
				case err == nil:
					res.QuotationLinkedPO = toLinkCheckPO(linked, nil)
				case !errors.Is(err, sql.ErrNoRows):
					return nil, err
				}
			}
		case !errors.Is(err, sql.ErrNoRows):
			return nil, err
		}
	}

	// 2. PO
	var po *models.PO
	if poNo != "" {
		p, err := s.poRepo.GetByOrderNum(ctx, poNo)
		switch {
		case err == nil:
			po = p
			quotations, err := s.quotationRepo.ListByPO(ctx, p.ID)
			if err != nil {
				return nil, err
			}
			nos := make([]string, 0, len(quotations))
			for _, q := range quotations {
				nos = append(nos, q.No)
			}
			res.PO = toLinkCheckPO(p, nos)
		case !errors.Is(err, sql.ErrNoRows):
			return nil, err
		}
	}

	// 3. Putusan link
	switch {
	case poNo != "" && po == nil:
		res.Level, res.Code = LinkLevelError, "po_not_found"
		res.Message = fmt.Sprintf("Nomor PO %s tidak ditemukan di RMS.", poNo)
	case po != nil && !IsLinkablePOStatus(po.Status):
		res.Level, res.Code = LinkLevelError, "po_not_linkable"
		res.Message = fmt.Sprintf("PO %s berstatus %s. Nomor PO hanya dapat dipakai bila PO berstatus prepared, progress, atau complete.", po.OrderNum, po.Status)
	case po != nil && res.QuotationLinkedPO != nil && res.QuotationLinkedPO.ID != po.ID:
		res.Level, res.Code = LinkLevelError, "linked_other_po"
		res.Message = fmt.Sprintf("Quotation %s sudah ter-link ke PO %s, bukan PO %s.",
			norm, res.QuotationLinkedPO.OrderNum, po.OrderNum)
	case po != nil && res.QuotationLinkedPO != nil:
		res.Level, res.Code = LinkLevelOK, "linked_match"
		res.Message = fmt.Sprintf("Quotation %s sudah ter-link ke PO %s.", norm, po.OrderNum)
	case po != nil && norm != "":
		res.Level, res.Code = LinkLevelWarning, "not_linked"
		res.Message = fmt.Sprintf("Quotation %s belum ter-link ke PO %s. Nomor PO tetap disimpan pada PR ini; link grup quotation dilakukan dari tab Quotation pada halaman PO (status prepared, progress, atau complete) setelah ada PR berstatus submitted, approved, atau completed di grup ini.", norm, po.OrderNum)
	case po != nil:
		res.Level, res.Code = LinkLevelInfo, "po_only"
		res.Message = fmt.Sprintf("Nomor PO %s akan disimpan pada PR ini. Isi nomor quotation bila PR ini bagian dari grup quotation.", po.OrderNum)
	case res.QuotationLinkedPO != nil:
		res.Level, res.Code = LinkLevelOK, "quotation_linked"
		res.Message = fmt.Sprintf("Quotation %s ter-link ke PO %s, nomor PO diisi otomatis.", norm, res.QuotationLinkedPO.OrderNum)
	default:
		res.Level, res.Code = LinkLevelInfo, "not_linked"
		res.Message = fmt.Sprintf("Quotation %s belum ter-link ke PO manapun.", norm)
	}

	switch {
	case res.QuotationLinkedPO != nil:
		res.PoNoDisplay = res.QuotationLinkedPO.OrderNum
	case po != nil && IsLinkablePOStatus(po.Status):
		res.PoNoDisplay = po.OrderNum
	default:
		res.PoNoDisplay = PoNotReleasedLabel
	}

	// 4. Lock responsible
	lock, err := s.resolveResponsibleLock(ctx, res.QuotationExists, res.QuotationID, po, in.ExcludePRID)
	if err != nil {
		return nil, err
	}
	res.Responsible = lock
	return res, nil
}

func (s *PRService) resolveResponsibleLock(ctx context.Context, quotationExists bool, quotationID int, po *models.PO, excludePRID int) (*ResponsibleLock, error) {
	build := func(source string, items []repository.QuotationResponsible) *ResponsibleLock {
		switch {
		case len(items) == 1:
			return &ResponsibleLock{Locked: true, Source: source, Items: items}
		case len(items) > 1:
			return &ResponsibleLock{Conflict: true, Source: source, Items: items}
		}
		return nil
	}

	if quotationExists {
		items, err := s.quotationRepo.ResponsiblesByQuotation(ctx, quotationID, excludePRID)
		if err != nil {
			return nil, err
		}
		if l := build("quotation", items); l != nil {
			return l, nil
		}
	}
	if po != nil {
		items, err := s.quotationRepo.ResponsiblesByPO(ctx, po.ID, excludePRID)
		if err != nil {
			return nil, err
		}
		return build("po", items), nil
	}
	return nil, nil
}

// ensureResponsibleMatchesQuotation: penegakan di sisi server. Kalau anggota grup
// memakai tepat satu responsible, PR yang join/edit harus memakai responsible yang sama.
// Grup lama yang sudah campur (>1 responsible) tidak dipaksa.
func (s *PRService) ensureResponsibleMatchesQuotation(ctx context.Context, quotationID, refResponsible, excludePRID int) error {
	items, err := s.quotationRepo.ResponsiblesByQuotation(ctx, quotationID, excludePRID)
	if err != nil {
		return err
	}
	if len(items) == 1 && items[0].ID != refResponsible {
		return ErrResponsibleMismatchWithQuotation
	}
	return nil
}