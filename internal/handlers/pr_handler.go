package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"fmt"

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
	exportService 	*service.PRExportService
	paymentRepo *repository.PRPaymentRepo
}

func NewPRHandler(
	repo *repository.PRRepo,
	historyRepo *repository.PRHistoryRepo,
	approvalRepo *repository.PRApprovalRepo,
	commentRepo *repository.PRCommentRepo,
	prService *service.PRService,
	approvalService *service.PRApprovalService,
	exportService *service.PRExportService,
) *PRHandler {
	return &PRHandler{
		repo:            repo,
		historyRepo:     historyRepo,
		approvalRepo:    approvalRepo,
		commentRepo:     commentRepo,
		prService:       prService,
		approvalService: approvalService,
		exportService: 	exportService,
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
		priorityDate := ""
		if p.PriorityDate != "" {
			priorityDate = utils.ParseDateParam(p.PriorityDate)
			if priorityDate == "" {
				utils.Error(w, http.StatusBadRequest, "Tanggal prioritas salah satu payment tidak valid")
				return
			}
		}
		payments = append(payments, repository.PRPaymentDraft{
			Stage: p.Stage, Amount: float64(p.Amount), Type: p.Type,
			Bank: p.Bank, BankAccountNo: p.BankAccountNo, BankAccountName: p.BankAccountName,
			PriorityDate: priorityDate,
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
		RefPreviousPR:     req.RefPreviousPRPtr(),
		Payments:          payments,
	}

	id, err := h.prService.CreatePR(r.Context(), adminID, in)
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
	utils.OK(w, "Fetch success", dto.NewPRResponse(*pr))
}

func (h *PRHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")

	if status != "" {
		data, err := h.repo.ListByStatus(r.Context(), []string{status})
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

	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
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

	if req.Payments != nil { // hanya proses kalau field disertakan di body
		drafts := make([]repository.PRPaymentDraft, 0, len(req.Payments))
		for _, p := range req.Payments {
			priorityDate := ""
			if p.PriorityDate != "" {
				priorityDate = utils.ParseDateParam(p.PriorityDate)
				if priorityDate == "" {
					utils.Error(w, http.StatusBadRequest, "Tanggal prioritas salah satu payment tidak valid")
					return
				}
			}
			drafts = append(drafts, repository.PRPaymentDraft{
				Stage: p.Stage, Amount: float64(p.Amount), Type: p.Type,
				Bank: p.Bank, BankAccountNo: p.BankAccountNo, BankAccountName: p.BankAccountName,
				PriorityDate: priorityDate,
			})
		}
		if err := h.paymentRepo.ReplaceDraftPayments(r.Context(), id, adminID, drafts); err != nil {
			utils.Error(w, http.StatusInternalServerError, "PR diperbarui, namun gagal memperbarui rencana pembayaran")
			return
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
	}
	if err := h.repo.Update(r.Context(), id, in); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui purchase request")
		return
	}
	utils.OK(w, "Purchase request berhasil diperbarui", nil)
}

func (h *PRHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
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

	approvers := make([]service.ApproverAssignment, 0, len(req.Approvers))
	for _, a := range req.Approvers {
		approvers = append(approvers, service.ApproverAssignment{
			Level: int(a.Level), Type: a.Type, AdminID: a.AdminID,
		})
	}

	err := h.approvalService.SubmitForApproval(r.Context(), id, adminID, approvers)
	switch {
	case err == nil:
		utils.OK(w, "Purchase request berhasil disubmit untuk approval", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRNotEligibleForSubmission):
		utils.Error(w, http.StatusConflict, "Purchase request harus berstatus draft atau revision sebelum dapat disubmit")
	case errors.Is(err, service.ErrApprovalLevelAlreadyExists):
		utils.Error(w, http.StatusConflict, "Level approval ini sudah dibuat untuk round yang sama")
	case errors.Is(err, service.ErrApprovalNoApprovers):
		utils.Error(w, http.StatusBadRequest, "Minimal satu approver wajib ditentukan")
	case errors.Is(err, service.ErrRequesterSignatureRequired): // BARU
		utils.Error(w, http.StatusConflict, "Anda belum memiliki tanda tangan terdaftar. Unggah tanda tangan terlebih dahulu melalui profil Anda sebelum dapat men-submit purchase request.")
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
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal memproses keputusan approval")
	}
}

func (h *PRHandler) SetPriority(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.SetPriorityRequest
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

	err := h.approvalService.SetPriority(r.Context(), id, adminID, req.Priority)
	switch {
	case err == nil:
		utils.OK(w, "Priority purchase request berhasil diperbarui", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRNotOwnedByActor):
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat mengatur priority purchase request milik sendiri")
	case errors.Is(err, service.ErrPRDraftCannotPriority):
		utils.Error(w, http.StatusConflict, "Priority tidak dapat diatur selama purchase request masih draft")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui priority purchase request")
	}
}

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

	if err := h.repo.UpdateAmounts(r.Context(), id, float64(req.PoAmount), float64(req.Hpp), req.PoNo); err != nil {
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
func (h *PRHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CancelPRRequest
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

	err := h.prService.CancelPR(r.Context(), id, adminID, req.Notes)
	switch {
	case err == nil:
		utils.OK(w, "Purchase request berhasil dibatalkan", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRCancelNotOwner):
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat membatalkan purchase request milik sendiri")
	case errors.Is(err, service.ErrPRAlreadyFinalized):
		utils.Error(w, http.StatusConflict, "Purchase request sudah pada status final dan tidak dapat dibatalkan")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal membatalkan purchase request")
	}
}

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
		approversDTO := it.Approvers
		if len(approversDTO) == 0 {
			approversDTO = req.DefaultApprovers
		}
		approvers := make([]service.ApproverAssignment, 0, len(approversDTO))
		for _, a := range approversDTO {
			approvers = append(approvers, service.ApproverAssignment{
				Level: int(a.Level), Type: a.Type, AdminID: a.AdminID,
			})
		}
		items = append(items, service.BulkSubmitItem{PRID: int(it.PRID), Approvers: approvers})
	}

	results := h.approvalService.BulkSubmitForApproval(r.Context(), adminID, items)

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