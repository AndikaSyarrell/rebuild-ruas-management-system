package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"strings"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
)

var (
	ErrPRInvalidStatus               = errors.New("status pr tidak valid")
	ErrPRNotApproved                 = errors.New("pr harus berstatus approved sebelum dapat masuk siklus pembayaran")
	ErrRequesterSignatureRequired    = errors.New("admin belum memiliki tanda tangan terdaftar, silakan unggah tanda tangan terlebih dahulu sebelum dapat membuat purchase request")
	ErrAdminHasNoDivision            = errors.New("admin belum terhubung ke divisi manapun, tidak dapat membuat nomor RFP")
	ErrPRAlreadyFinalized            = errors.New("purchase request sudah berada pada status final (approved/completed/rejected/cancelled) dan tidak dapat dibatalkan")
	ErrPRCancelNotOwner              = errors.New("hanya requester pemilik purchase request yang dapat membatalkannya")
	ErrPreviousPRNotFound            = errors.New("purchase request referensi tidak ditemukan")
	ErrPreviousPRNotOwned            = errors.New("purchase request referensi bukan milik admin yang sama")
	ErrPreviousPRCrossResponsible    = errors.New("purchase request referensi harus memiliki responsible (CoA) yang sama")
	ErrPreviousPRCircular            = errors.New("referensi purchase request tidak boleh membentuk siklus")
	ErrQuotationConfirmationRequired = errors.New("nomor quotation sudah dipakai oleh grup purchase request lain, konfirmasi diperlukan untuk bergabung")
	ErrQuotationMismatchWithPrevious = errors.New("qout_no yang diisi berbeda dengan nomor quotation milik purchase request sebelumnya pada chain ini")
	ErrPRAlreadyLinkedToPO           = errors.New("purchase request ini sudah ter-link ke PO lewat quotation, ubah po_no melalui proses linking PO, bukan lewat endpoint ini")
	ErrQuotationLocked               = errors.New("nomor quotation tidak dapat diubah setelah purchase order")
	ErrPONotFound                = errors.New("nomor po tidak ditemukan di rms")
	ErrPONoMismatchWithQuotation = errors.New("nomor po berbeda dengan po yang ter-link ke grup quotation ini")
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
const maxQuotationSuggestions = 10
const PoNotReleasedLabel = "PO BELUM RELEASE"

type PRService struct {
	prRepo        *repository.PRRepo
	historyRepo   *repository.PRHistoryRepo
	paymentRepo   *repository.PRPaymentRepo
	signatureRepo *repository.AdminSignatureRepo
	adminRepo     *repository.AdminRepo
	divisionRepo  *repository.DivisionRepo
	counterRepo   *repository.PRCounterRepo
	quotationRepo *repository.PRQuotationRepo // BARU - PR-PO Linking
	poRepo        *repository.PORepo          // BARU - hanya dipakai untuk baca po_order_num saat sync BR-LINK-03
}

func NewPRService(
	prRepo *repository.PRRepo,
	historyRepo *repository.PRHistoryRepo,
	paymentRepo *repository.PRPaymentRepo,
	signatureRepo *repository.AdminSignatureRepo,
	adminRepo *repository.AdminRepo,
	divisionRepo *repository.DivisionRepo,
	counterRepo *repository.PRCounterRepo,
	quotationRepo *repository.PRQuotationRepo, // BARU
	poRepo *repository.PORepo, // BARU
) *PRService {
	return &PRService{
		prRepo: prRepo, historyRepo: historyRepo, paymentRepo: paymentRepo,
		signatureRepo: signatureRepo, adminRepo: adminRepo,
		divisionRepo: divisionRepo, counterRepo: counterRepo,
		quotationRepo: quotationRepo, poRepo: poRepo,
	}
}

func (s *PRService) SearchQuotations(ctx context.Context, keyword string) ([]repository.QuotationCandidate, error) {
	quotations, err := s.quotationRepo.SearchByNo(ctx, keyword, maxQuotationSuggestions)
	if err != nil {
		return nil, err
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

func (s *PRService) CreatePR(ctx context.Context, adminID string, in repository.PRInput, confirmJoinQuotation bool) (int64, *QuotationConflict, error) {
	if in.RefPreviousPR != nil {
		if err := s.ValidatePreviousPRReference(ctx, adminID, in.RefResponsible, 0, *in.RefPreviousPR); err != nil {
			return 0, nil, err
		}
	}

	quotationID, conflict, err := s.resolveQuotation(ctx, in.QoutNo, in.RefPreviousPR, confirmJoinQuotation)
	if err != nil {
		return 0, conflict, err
	}
	if quotationID != nil {
		if err := s.ensureResponsibleMatchesQuotation(ctx, *quotationID, in.RefResponsible, 0); err != nil {
			return 0, nil, err
		}
	}

	typedPoNo := ""
	if in.PoNo != nil {
		typedPoNo = *in.PoNo
	}
	poNo, err := s.resolvePoNo(ctx, typedPoNo, quotationID, in.RefPreviousPR)
	if err != nil {
		return 0, nil, err
	}
	in.PoNo = &poNo

	rfpNo, err := s.generateRfpNo(ctx, adminID)
	if err != nil {
		return 0, nil, err
	}
	in.RefAdmin = adminID
	in.RfpNo = rfpNo
	in.SignatureRef = 0
	in.RefQuotation = quotationID

	prID, err := s.prRepo.Create(ctx, in)
	if err != nil {
		return 0, nil, err
	}
	return prID, nil, nil
}

// func (s *PRService) syncPoNoFromQuotation(ctx context.Context, prID, quotationID int) {
// 	q, err := s.quotationRepo.GetByID(ctx, quotationID)
// 	if err != nil || q.RefPO == nil {
// 		return
// 	}
// 	po, err := s.poRepo.GetDetail(ctx, *q.RefPO)
// 	if err != nil {
// 		log.Printf("gagal membaca po %s untuk pr %d: %v", *q.RefPO, prID, err)
// 		return
// 	}
// 	if err := s.prRepo.SyncPoNo(ctx, prID, po.OrderNum); err != nil {
// 		log.Printf("gagal sync pr_po_no pr %d: %v", prID, err)
// 	}
// }

func (s *PRService) UpdatePR(ctx context.Context, prID int, in repository.PRInput, confirmJoinQuotation bool) (*QuotationConflict, error) {
	current, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	norm := repository.NormalizeQuotationNo(in.QoutNo)

	if current.RefQuotation != nil {
		q, err := s.quotationRepo.GetByID(ctx, *current.RefQuotation)
		if err != nil {
			return nil, err
		}
		if norm != "" && norm != q.No {
			return nil, ErrQuotationLocked
		}
		// ganti ref_previous_pr ke chain dengan quotation lain = chain terbelah
		if in.RefPreviousPR != nil {
			prev, err := s.prRepo.GetByID(ctx, *in.RefPreviousPR)
			if err != nil {
				return nil, err
			}
			if prev.RefQuotation != nil && *prev.RefQuotation != *current.RefQuotation {
				return nil, ErrQuotationMismatchWithPrevious
			}
		}
		in.RefQuotation = current.RefQuotation
		in.QoutNo = q.No
	} else {
		qid, conflict, err := s.resolveQuotation(ctx, in.QoutNo, in.RefPreviousPR, confirmJoinQuotation)
		if err != nil {
			return conflict, err
		}
		in.RefQuotation = qid
	}

	if in.RefQuotation != nil {
		if err := s.ensureResponsibleMatchesQuotation(ctx, *in.RefQuotation, in.RefResponsible, prID); err != nil {
			return nil, err
		}
	}

	newlyJoined := current.RefQuotation == nil && in.RefQuotation != nil
	if in.PoNo != nil || newlyJoined {
		typed := ""
		if in.PoNo != nil {
			typed = *in.PoNo
		}
		resolved, err := s.resolvePoNo(ctx, typed, in.RefQuotation, in.RefPreviousPR)
		if err != nil {
			return nil, err
		}
		if in.PoNo != nil || resolved != "" {
			in.PoNo = &resolved
		} else {
			in.PoNo = nil // bergabung ke grup tanpa PO: jangan menimpa nilai yang sudah ada
		}
	}

	if err := s.prRepo.Update(ctx, prID, in); err != nil {
		return nil, err
	}
	return nil, nil
}

type QuotationConflict struct {
	QuotationID int
	QuotationNo string
	LinkedPO    string // "" bila grup ini belum ter-link ke PO manapun
	Members     []repository.QuotationMember
}

func (s *PRService) resolveQuotation(ctx context.Context, qoutNo string, refPreviousPR *int, confirmJoin bool) (*int, *QuotationConflict, error) {
	norm := repository.NormalizeQuotationNo(qoutNo)

	if refPreviousPR != nil {
		prev, err := s.prRepo.GetByID(ctx, *refPreviousPR)
		if err != nil {
			return nil, nil, err
		}
		if prev.RefQuotation != nil {
			if norm != "" {
				prevQuotation, err := s.quotationRepo.GetByID(ctx, *prev.RefQuotation)
				if err != nil {
					return nil, nil, err
				}
				if norm != prevQuotation.No {
					return nil, nil, ErrQuotationMismatchWithPrevious
				}
			}
			return prev.RefQuotation, nil, nil
		}
		// PR sebelumnya belum punya quotation (mis. dibuat sebelum fitur ini
		// aktif) - jatuh ke resolusi normal di bawah berdasarkan norm.
	}

	if norm == "" {
		return nil, nil, nil
	}

	existing, err := s.quotationRepo.GetByNo(ctx, norm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			newID, err := s.quotationRepo.Create(ctx, norm)
			if err != nil {
				return nil, nil, err
			}
			id := int(newID)
			return &id, nil, nil
		}
		return nil, nil, err
	}

	if !confirmJoin {
		members, err := s.quotationRepo.MemberPRs(ctx, existing.ID)
		if err != nil {
			return nil, nil, err
		}
		linkedPO := ""
		if existing.RefPO != nil {
			linkedPO = *existing.RefPO
		}
		return nil, &QuotationConflict{
			QuotationID: existing.ID,
			QuotationNo: existing.No,
			LinkedPO:    linkedPO,
			Members:     members,
		}, ErrQuotationConfirmationRequired
	}

	return &existing.ID, nil, nil
}

func (s *PRService) UpdateAmounts(ctx context.Context, prID int, poAmount, hpp float64, poNo string) error {
	if poNo != "" {
		pr, err := s.prRepo.GetByID(ctx, prID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if pr.RefQuotation != nil {
			q, err := s.quotationRepo.GetByID(ctx, *pr.RefQuotation)
			if err != nil {
				return err
			}
			if q.RefPO != nil {
				return ErrPRAlreadyLinkedToPO
			}
		}
	}
	return s.prRepo.UpdateAmounts(ctx, prID, poAmount, hpp, poNo)
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

func (s *PRService) resolvePoNo(ctx context.Context, typedPoNo string, quotationID *int, refPreviousPR *int) (string, error) {
	typed := strings.TrimSpace(typedPoNo)

	// 1. grup quotation sudah ter-link ke PO -> nomor itu, walau PR masih draft
	if quotationID != nil {
		q, err := s.quotationRepo.GetByID(ctx, *quotationID)
		if err != nil {
			return "", err
		}
		if q.RefPO != nil {
			po, err := s.poRepo.GetDetail(ctx, *q.RefPO)
			if err != nil {
				return "", err
			}
			if typed != "" && !strings.EqualFold(typed, po.OrderNum) {
				return "", ErrPONoMismatchWithQuotation
			}
			return po.OrderNum, nil
		}
	}

	// 2. nomor PO yang diisi requester
	if typed != "" {
		po, err := s.poRepo.GetByOrderNum(ctx, typed)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrPONotFound
			}
			return "", err
		}
		if !IsLinkablePOStatus(po.Status) {
			return "", ErrPONotLinkable
		}
		return po.OrderNum, nil
	}

	// 3. warisi dari PR sebelumnya pada chain
	if refPreviousPR != nil {
		prev, err := s.prRepo.GetByID(ctx, *refPreviousPR)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", nil
			}
			return "", err
		}
		if prev.PoNo != nil {
			return strings.TrimSpace(*prev.PoNo), nil
		}
	}

	// 4. belum ada -> kosong (NULL), tampil "PO BELUM RELEASE"
	return "", nil
}