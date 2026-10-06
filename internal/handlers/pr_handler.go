package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type PRHandler struct {
	repo            *repository.PRRepo
	historyRepo     *repository.PRHistoryRepo
	approvalRepo    *repository.PRApprovalRepo
	commentRepo     *repository.PRCommentRepo
	prService       *service.PRService
	approvalService *service.PRApprovalService
	exportService   *service.PRExportService
	paymentRepo     *repository.PRPaymentRepo
}

func NewPRHandler(
	repo *repository.PRRepo,
	historyRepo *repository.PRHistoryRepo,
	approvalRepo *repository.PRApprovalRepo,
	commentRepo *repository.PRCommentRepo,
	prService *service.PRService,
	approvalService *service.PRApprovalService,
	exportService *service.PRExportService,
	paymentRepo *repository.PRPaymentRepo,
) *PRHandler {
	return &PRHandler{
		repo:            repo,
		historyRepo:     historyRepo,
		approvalRepo:    approvalRepo,
		commentRepo:     commentRepo,
		prService:       prService,
		approvalService: approvalService,
		exportService:   exportService,
		paymentRepo:     paymentRepo,
	}
}

func (h *PRHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePRRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	targetInvoiceDate := ""
	if req.TargetInvoiceDate != "" {
		targetInvoiceDate = utils.ParseDateParam(req.TargetInvoiceDate)
		if targetInvoiceDate == "" {
			utils.Error(w, http.StatusBadRequest, "Tanggal target invoice tidak valid")
			return
		}
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	payments := make([]repository.PRPaymentDraft, 0, len(req.Payments))
	for _, p := range req.Payments {
		payments = append(payments, repository.PRPaymentDraft{
			Stage: p.Stage, Amount: float64(p.Amount), Type: p.Type,
			Bank: p.Bank, BankAccountNo: p.BankAccountNo, BankAccountName: p.BankAccountName,
		})
	}

	in := repository.PRInput{
		RefResponsible:    int(req.RefResponsible),
		DescriptionItem:   req.DescriptionItem,
		SubClient:         req.SubClient,
		RequestedAmount:   float64(req.RequestedAmount),
		PoAmount:          float64(req.PoAmount),
		Hpp:               float64(req.Hpp),
		QoutNo:            req.QoutNo,
		TargetInvoiceDate: targetInvoiceDate,
		Priority:      	   req.Priority,
		RefPreviousPR: 	   req.RefPreviousPRPtr(),
		PoNo: 			   &req.PoNo,
		Payments:      	   payments,
	}

	id, conflict, err := h.prService.CreatePR(r.Context(), adminID, in, req.ConfirmJoinQuotation)
	switch {
	case err == nil:
		utils.Created(w, "Purchase request berhasil dibuat", map[string]any{"pr_id": id})
	case errors.Is(err, service.ErrAdminHasNoDivision):
		utils.Error(w, http.StatusConflict, "Akun Anda belum terhubung ke divisi manapun, hubungi administrator untuk mengatur divisi sebelum membuat purchase request.")
	case errors.Is(err, service.ErrPreviousPRNotFound):
		utils.Error(w, http.StatusBadRequest, "Purchase request referensi tidak ditemukan")
	case errors.Is(err, service.ErrPreviousPRNotOwned):
		utils.Error(w, http.StatusForbidden, "Purchase request referensi bukan milik Anda")
	case errors.Is(err, service.ErrPreviousPRCrossResponsible):
		utils.Error(w, http.StatusBadRequest, "Purchase request referensi harus memiliki responsible (CoA) yang sama")
	case errors.Is(err, service.ErrPreviousPRCircular):
		utils.Error(w, http.StatusBadRequest, "Referensi tidak boleh membentuk siklus")
	case errors.Is(err, service.ErrQuotationMismatchWithPrevious):
		utils.Error(w, http.StatusBadRequest, "Nomor quotation (qout_no) berbeda dengan quotation milik purchase request sebelumnya pada chain ini")
	case errors.Is(err, service.ErrQuotationConfirmationRequired):
		utils.JSON(w, http.StatusConflict, false,"Nomor quotation ini sudah dipakai oleh grup purchase request lain. Konfirmasi untuk bergabung ke grup tersebut.", dto.NewQuotationConflictResponse(conflict))
	case errors.Is(err, service.ErrResponsibleMismatchWithQuotation):
		utils.Error(w, http.StatusConflict, "Responsible (CoA) harus sama dengan responsible pada grup quotation ini")
	case errors.Is(err, service.ErrPONotFound):
		utils.Error(w, http.StatusBadRequest, "Nomor PO tidak ditemukan di RMS")
	case errors.Is(err, service.ErrPONotLinkable):
		utils.Error(w, http.StatusConflict, "Nomor PO hanya dapat dipakai bila PO berstatus prepared, progress, atau complete")
	case errors.Is(err, service.ErrPONoMismatchWithQuotation):
		utils.Error(w, http.StatusConflict, "Nomor PO berbeda dengan PO yang ter-link ke grup quotation ini")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat purchase request")
	}
}

func (h *PRHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	pr, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		return
	}
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}
	if !requirePRVisible(w, r, h.repo, id) {
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRResponse(*pr))
}

func (h *PRHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	adminID, ok := actorFromContext(r.Context())

	if status != "" {
		data, err := h.repo.ListByStatus(r.Context(), []string{status}, adminID)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
			return
		}
		utils.OK(w, "Fetch success", map[string]any{
			"paged": false,
			"data":  dto.NewPRResponseList(data),
		})
		return
	}
	
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewPRResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *PRHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdatePRRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	targetInvoiceDate := ""
	if req.TargetInvoiceDate != "" {
		targetInvoiceDate = utils.ParseDateParam(req.TargetInvoiceDate)
		if targetInvoiceDate == "" {
			utils.Error(w, http.StatusBadRequest, "Tanggal target invoice tidak valid")
			return
		}
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	current, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		return
	}
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}
	if current.RefAdmin != adminID {
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat mengubah purchase request milik sendiri")
		return
	}
	if current.Status != "draft" && current.Status != "revision" {
		utils.Error(w, http.StatusConflict, "Purchase request hanya dapat diubah saat berstatus draft atau revision")
		return
	}

	if req.RefPreviousPR != nil {
		err := h.prService.ValidatePreviousPRReference(r.Context(), adminID, int(req.RefResponsible), id, int(*req.RefPreviousPR))
		switch {
		case err == nil:
		case errors.Is(err, service.ErrPreviousPRNotFound):
			utils.Error(w, http.StatusBadRequest, "Purchase request referensi tidak ditemukan")
			return
		case errors.Is(err, service.ErrPreviousPRNotOwned):
			utils.Error(w, http.StatusForbidden, "Purchase request referensi bukan milik Anda")
			return
		case errors.Is(err, service.ErrPreviousPRCrossResponsible):
			utils.Error(w, http.StatusBadRequest, "Purchase request referensi harus memiliki responsible (CoA) yang sama")
			return
		case errors.Is(err, service.ErrPreviousPRCircular):
			utils.Error(w, http.StatusBadRequest, "Referensi tidak boleh membentuk siklus dengan purchase request ini")
			return
		default:
			utils.Error(w, http.StatusInternalServerError, "Gagal memvalidasi referensi purchase request")
			return
		}
	}

	var drafts []repository.PRPaymentDraft
	if req.Payments != nil {
		drafts = make([]repository.PRPaymentDraft, 0, len(req.Payments))
		for _, p := range req.Payments {
			drafts = append(drafts, repository.PRPaymentDraft{
				Stage: p.Stage, Amount: float64(p.Amount), Type: p.Type,
				Bank: p.Bank, BankAccountNo: p.BankAccountNo, BankAccountName: p.BankAccountName,
			})
		}
	}

	in := repository.PRInput{
		RefResponsible:    int(req.RefResponsible),
		DescriptionItem:   req.DescriptionItem,
		SubClient:         req.SubClient,
		RequestedAmount:   float64(req.RequestedAmount),
		QoutNo:            req.QoutNo,
		TargetInvoiceDate: targetInvoiceDate,
		RefPreviousPR:     req.RefPreviousPRPtr(),
		PoNo: 			   req.PoNo,
		PoAmount: 		   float64(req.PoAmount),
		Hpp:      		   float64(req.Hpp),
	}
	conflict, err := h.prService.UpdatePR(r.Context(), id, in, req.ConfirmJoinQuotation)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrResponsibleMismatchWithQuotation):
			utils.Error(w, http.StatusConflict, "Responsible (CoA) harus sama dengan responsible pada grup quotation ini")
		case errors.Is(err, service.ErrPONotFound):
			utils.Error(w, http.StatusBadRequest, "Nomor PO tidak ditemukan di RMS")
		case errors.Is(err, service.ErrPONotLinkable):
			utils.Error(w, http.StatusConflict, "Nomor PO hanya dapat dipakai bila PO berstatus prepared, progress, atau complete")
		case errors.Is(err, service.ErrPONoMismatchWithQuotation):
			utils.Error(w, http.StatusConflict, "Nomor PO berbeda dengan PO yang ter-link ke grup quotation ini")
		case errors.Is(err, service.ErrNotFound):
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		case errors.Is(err, service.ErrQuotationConfirmationRequired):
			utils.JSON(w, http.StatusConflict, false, "Nomor quotation ini sudah dipakai oleh grup purchase request lain. Konfirmasi untuk bergabung ke grup tersebut.", dto.NewQuotationConflictResponse(conflict))
		case errors.Is(err, service.ErrQuotationLocked):
			utils.Error(w, http.StatusConflict, "Nomor quotation tidak dapat diubah karena purchase request ini sudah tergabung dalam grup quotation")
		case errors.Is(err, service.ErrQuotationMismatchWithPrevious):
			utils.Error(w, http.StatusBadRequest, "Nomor quotation (qout_no) berbeda dengan quotation milik purchase request sebelumnya pada chain ini")
		default:
			utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui purchase request")
		}
		return
	}

	if drafts != nil {
		if err := h.paymentRepo.ReplaceDraftPayments(r.Context(), id, adminID, drafts); err != nil {
			utils.Error(w, http.StatusInternalServerError, "PR diperbarui, namun gagal memperbarui rencana pembayaran")
			return
		}
	}
	utils.OK(w, "Purchase request berhasil diperbarui", nil)
}

func (h *PRHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !requirePRVisible(w, r, h.repo, id) {
		return
	}
	if !ok {
		return
	}
	data, err := h.historyRepo.ListByPR(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil riwayat status purchase request")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRHistoryResponseList(data))
}

func (h *PRHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !requirePRVisible(w, r, h.repo, id) {
		return
	}
	if !ok {
		return
	}
	data, err := h.approvalRepo.ListByPR(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar approval purchase request")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRApprovalResponseList(data))
}

func (h *PRHandler) Submit(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.SubmitPRRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	err := h.approvalService.SubmitForApproval(r.Context(), id, adminID, int(req.SignatureID), strings.TrimSpace(req.CheckerID))
	switch {
	case err == nil:
		utils.OK(w, "Purchase request berhasil disubmit untuk approval", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRNotEligibleForSubmission):
		utils.Error(w, http.StatusConflict, "Purchase request harus berstatus draft atau revision sebelum dapat disubmit")
	case errors.Is(err, service.ErrCheckerInvalid):
		utils.Error(w, http.StatusBadRequest, "Checker yang dipilih tidak aktif atau tidak memiliki akses checker")
	case errors.Is(err, service.ErrCheckerIsRequester):
		utils.Error(w, http.StatusBadRequest, "Anda tidak dapat menjadi checker untuk purchase request Anda sendiri")
	case errors.Is(err, service.ErrSignatureNotOwnedByActor):
		utils.Error(w, http.StatusForbidden, "Tanda tangan yang dipilih bukan milik Anda")
	case errors.Is(err, service.ErrRequesterSignatureRequired):
		utils.Error(w, http.StatusConflict, "Anda belum memiliki tanda tangan terdaftar. Buat tanda tangan terlebih dahulu sebelum men-submit purchase request.")
	case errors.Is(err, service.ErrPRNotOwnedByActor):
		utils.Error(w, http.StatusForbidden, "Purchase request ini bukan milik admin yang login")
	case errors.Is(err, service.ErrCostControlAttachmentRequired):
		utils.Error(w, http.StatusConflict, "Purchase request dengan nominal di atas Rp 50.000.000 wajib melampirkan dokumen cost control terlebih dahulu sebelum dapat disubmit")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal submit purchase request")
	}
}

func (h *PRHandler) ApproveApproval(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "approved")
}

func (h *PRHandler) RejectApproval(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "rejected")
}

func (h *PRHandler) decide(w http.ResponseWriter, r *http.Request, decision string) {
	approvalID, ok := utils.ParseIDParam(w, chi.URLParam(r, "approvalId"))
	if !ok {
		return
	}
	var req dto.DecidePRApprovalRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	err := h.approvalService.Decide(r.Context(), approvalID, adminID, decision, req.Notes)
	switch {
	case err == nil:
		msg := "Approval berhasil disetujui"
		if decision == "rejected" {
			msg = "Approval berhasil ditolak"
		}
		utils.OK(w, msg, nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Baris approval tidak ditemukan")
	case errors.Is(err, service.ErrApprovalNotOwnedByActor):
		utils.Error(w, http.StatusForbidden, "Baris approval ini bukan milik admin yang login")
	case errors.Is(err, service.ErrApprovalSequenceViolation):
		utils.Error(w, http.StatusConflict, "Approval level sebelumnya belum disetujui")
	case errors.Is(err, service.ErrPRNotSubmittedForDecision):
		utils.Error(w, http.StatusConflict, "Purchase request ini sudah tidak berstatus submitted (mungkin sudah diputuskan/diminta revisi oleh level lain)")
	case errors.Is(err, service.ErrInvalidApprovalDecision):
		utils.Error(w, http.StatusBadRequest, "Keputusan approval harus approved atau rejected")
	case errors.Is(err, service.ErrApproverSignatureRequired):
		utils.Error(w, http.StatusConflict, "Anda belum memiliki tanda tangan terdaftar. Unggah tanda tangan terlebih dahulu melalui profil Anda sebelum dapat melakukan approval.")
	case errors.Is(err, service.ErrApprovalAlreadyDecided):
		utils.Error(w, http.StatusConflict, "Baris approval ini sudah diputuskan")
	case errors.Is(err, service.ErrRejectNotAllowedForLevel):
		utils.Error(w, http.StatusForbidden, "Reject hanya dapat dilakukan pada tahap checker")
	case errors.Is(err, service.ErrApproverAccessRequired):
		utils.Error(w, http.StatusForbidden, "Anda tidak memiliki akses untuk tahap approval ini")
	case errors.Is(err, service.ErrApproverIsRequester):
		utils.Error(w, http.StatusForbidden, "Anda tidak dapat menyetujui purchase request milik sendiri")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal memproses keputusan approval")
	}
}

// func (h *PRHandler) SetPriority(w http.ResponseWriter, r *http.Request) {
// 	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
// 	if !ok {
// 		return
// 	}
// 	var req dto.SetPriorityRequest
// 	if err := decodeJSON(r, &req); err != nil {
// 		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
// 		return
// 	}
// 	if err := req.Validate(); err != nil {
// 		utils.Error(w, http.StatusBadRequest, err.Error())
// 		return
// 	}

// 	adminID, ok := actorFromContext(r.Context())
// 	if !ok {
// 		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
// 		return
// 	}

// 	err := h.approvalService.SetPriority(r.Context(), id, adminID, req.Priority)
// 	switch {
// 	case err == nil:
// 		utils.OK(w, "Priority purchase request berhasil diperbarui", nil)
// 	case errors.Is(err, service.ErrNotFound):
// 		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
// 	case errors.Is(err, service.ErrPRNotOwnedByActor):
// 		utils.Error(w, http.StatusForbidden, "Anda hanya dapat mengatur priority purchase request milik sendiri")
// 	case errors.Is(err, service.ErrPRDraftCannotPriority):
// 		utils.Error(w, http.StatusConflict, "Priority tidak dapat diatur selama purchase request masih draft")
// 	default:
// 		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui priority purchase request")
// 	}
// }

func (h *PRHandler) RequestRevision(w http.ResponseWriter, r *http.Request) {
	approvalID, ok := utils.ParseIDParam(w, chi.URLParam(r, "approvalId"))
	if !ok {
		return
	}
	var req dto.DecidePRApprovalRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	err := h.approvalService.RequestRevision(r.Context(), approvalID, adminID, req.Notes)
	switch {
	case err == nil:
		utils.OK(w, "Permintaan revisi berhasil dikirim", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Baris approval tidak ditemukan")
	case errors.Is(err, service.ErrApprovalNotOwnedByActor):
		utils.Error(w, http.StatusForbidden, "Baris approval ini bukan milik admin yang login")
	case errors.Is(err, service.ErrRevisionNotAllowedForLevel):
		utils.Error(w, http.StatusForbidden, "Approver level ini (director) tidak berwenang meminta revisi")
	case errors.Is(err, service.ErrPRNotSubmittedForDecision):
		utils.Error(w, http.StatusConflict, "Purchase request ini sudah tidak berstatus submitted")
	case errors.Is(err, service.ErrApprovalSequenceViolation):
		utils.Error(w, http.StatusConflict, "Approval level sebelumnya belum disetujui")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal memproses permintaan revisi")
	}
}

func (h *PRHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	related, err := h.repo.IsAdminRelated(r.Context(), id, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa keterkaitan dengan purchase request")
		return
	}
	if !related {
		utils.Error(w, http.StatusForbidden, "Anda tidak terkait dengan purchase request ini (bukan requester, approver, atau finance terkait)")
		return
	}

	data, err := h.commentRepo.ListByPR(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar komentar")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRCommentResponseList(data))
}

func (h *PRHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.PRCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	if _, err := h.repo.GetByID(r.Context(), id); err != nil {
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		return
	}

	related, err := h.repo.IsAdminRelated(r.Context(), id, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa keterkaitan dengan purchase request")
		return
	}
	if !related {
		utils.Error(w, http.StatusForbidden, "Anda tidak terkait dengan purchase request ini (bukan requester, approver, atau finance terkait)")
		return
	}

	if _, err := h.commentRepo.Insert(r.Context(), id, adminID, req.Text, req.Type); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan komentar")
		return
	}
	utils.Created(w, "Komentar berhasil ditambahkan", nil)
}

func (h *PRHandler) MyTurn(w http.ResponseWriter, r *http.Request) {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	data, err := h.approvalRepo.ListPendingByApprover(r.Context(), adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar approval yang menunggu")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRApprovalResponseList(data))
}

func (h *PRHandler) UpdateAmounts(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdatePRAmountsRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.repo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}

	if err := h.prService.UpdateAmounts(r.Context(), id, float64(req.PoAmount), float64(req.Hpp), req.PoNo); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrPRAlreadyLinkedToPO) {
			utils.Error(w, http.StatusConflict, "Purchase request ini sudah ter-link ke PO lewat quotation. Ubah nomor PO melalui proses linking PO, bukan lewat endpoint ini.")
			return
		}
		if errors.Is(err, service.ErrPONotFound) {
			utils.Error(w, http.StatusBadRequest, "Nomor PO tidak ditemukan di RMS")
			return
		}
		if errors.Is(err, service.ErrPONotLinkable) {
			utils.Error(w, http.StatusConflict, "Nomor PO hanya dapat dipakai bila PO berstatus prepared, progress, atau complete")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui nominal purchase request")
		return
	}

	updated, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Nominal tersimpan, namun gagal mengambil data terbaru")
		return
	}
	utils.OK(w, "Nominal purchase request berhasil diperbarui", dto.NewPRResponse(*updated))
}

func (h *PRHandler) ExportRFP(w http.ResponseWriter, r *http.Request) {
		id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if !requirePRVisible(w, r, h.repo, id) {
		return
	}
	data, filename, err := h.exportService.GenerateRFPDocx(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPRExportSourceMissing) {
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat dokumen RFP")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.Write(data)
}

// POST /api/pr/{id}/cancel

func (h *PRHandler) BulkSubmit(w http.ResponseWriter, r *http.Request) {
	var req dto.BulkSubmitPRRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	items := make([]service.BulkSubmitItem, 0, len(req.Items))
	for _, it := range req.Items {
		checkerID := strings.TrimSpace(it.CheckerID)
		if checkerID == "" {
			checkerID = strings.TrimSpace(req.DefaultCheckerID)
		}
		items = append(items, service.BulkSubmitItem{PRID: int(it.PRID), CheckerID: checkerID})
	}
	results := h.approvalService.BulkSubmitForApproval(r.Context(), adminID, int(req.SignatureID), items)

	successCount := 0
	for _, res := range results {
		if res.Success {
			successCount++
		}
	}
	msg := fmt.Sprintf("Bulk submit selesai: %d berhasil, %d gagal dari %d total",
		successCount, len(results)-successCount, len(results))
	utils.OK(w, msg, results)
}

// POST /api/pr/approvals/bulk-decide
func (h *PRHandler) BulkDecide(w http.ResponseWriter, r *http.Request) {
	var req dto.BulkDecideApprovalRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	items := make([]service.BulkDecideItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, service.BulkDecideItem{ApprovalID: int(it.ApprovalID), Notes: it.Notes})
	}

	results := h.approvalService.BulkDecide(r.Context(), adminID, req.Decision, items)

	successCount := 0
	for _, res := range results {
		if res.Success {
			successCount++
		}
	}
	actionLabel := "approve"
	if req.Decision == "rejected" {
		actionLabel = "reject"
	}
	msg := fmt.Sprintf("Bulk %s selesai: %d berhasil, %d gagal dari %d total",
		actionLabel, successCount, len(results)-successCount, len(results))
	utils.OK(w, msg, results)
}

func (h *PRHandler) SearchQuotations(w http.ResponseWriter, r *http.Request) {
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	if keyword == "" {
		utils.OK(w, "Fetch success", []dto.QuotationCandidateResponse{})
		return
	}
 
	list, err := h.prService.SearchQuotations(r.Context(), keyword)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mencari quotation")
		return
	}
	utils.OK(w, "Fetch success", dto.NewQuotationCandidateResponseList(list))
}

func (h *PRHandler) SearchPOOptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	keyword := strings.TrimSpace(q.Get("keyword"))
	limit := utils.AtoiDefault(q.Get("limit"), 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	list, err := h.prService.SearchPOOptions(r.Context(), keyword, limit)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar nomor PO")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPOLinkOptionResponseList(list))
}

// GET /api/pr/link-check?qout_no=&po_no=&exclude_pr_id=
// Hasil validasi ada di data.level (bukan status HTTP): 200 berarti pemeriksaan berhasil dijalankan.
func (h *PRHandler) CheckLink(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.prService.CheckQuotationPOLink(r.Context(), service.LinkCheckInput{
		QoutNo:      q.Get("qout_no"),
		PoNo:        q.Get("po_no"),
		ExcludePRID: utils.AtoiDefault(q.Get("exclude_pr_id"), 0),
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa link quotation dan PO")
		return
	}
	utils.OK(w, "Fetch success", dto.NewLinkCheckResponse(res))
}

var validDecisionFilter = map[string]bool{"approved": true, "rejected": true, "revision_requested": true}

// GET /api/pr/my-decisions?decision=&page=&item=
func (h *PRHandler) MyDecisions(w http.ResponseWriter, r *http.Request) {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	decision := strings.TrimSpace(r.URL.Query().Get("decision"))
	if decision != "" && !validDecisionFilter[decision] {
		utils.Error(w, http.StatusBadRequest, "decision harus salah satu dari: approved, rejected, revision_requested")
		return
	}

	p := utils.ParsePagination(r)
	data, total, err := h.approvalRepo.ListDecidedByApprover(r.Context(), adminID, decision, p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil riwayat keputusan approval")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewPRDecisionHistoryResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}