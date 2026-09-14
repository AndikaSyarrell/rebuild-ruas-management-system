package handlers

import (
	// "database/sql"
	"errors"
	"net/http"
	"strings"
	"fmt"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type PRPaymentHandler struct {
	repo    *repository.PRPaymentRepo
	prRepo  *repository.PRRepo
	service *service.PRPaymentService
	exportService *service.PRPaymentExportService
}

var validPaymentExportStatus = map[string]bool{"pending": true, "paid": true, "cancelled": true, "draft": true}

func NewPRPaymentHandler(repo *repository.PRPaymentRepo, prRepo *repository.PRRepo, svc *service.PRPaymentService, exportService *service.PRPaymentExportService) *PRPaymentHandler {
	return &PRPaymentHandler{repo: repo, prRepo: prRepo, service: svc, exportService: exportService}
}

// GET /api/pr/{id}/payments
func (h *PRPaymentHandler) ListByPR(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.ListByPR(r.Context(), prID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar pembayaran")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRPaymentResponseList(data))
}

// POST /api/pr/{id}/payments
func (h *PRPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreatePRPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	priorityDate := ""
	if req.PriorityDate != "" {
		priorityDate = utils.ParseDateParam(req.PriorityDate)
		if priorityDate == "" {
			utils.Error(w, http.StatusBadRequest, "Tanggal prioritas pembayaran tidak valid")
			return
		}
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	in := repository.PRPaymentInput{
		Stage: req.Stage, Amount: float64(req.Amount), Type: req.Type,
		Bank: req.Bank, BankAccountNo: req.BankAccountNo, BankAccountName: req.BankAccountName,
		PriorityDate: priorityDate,
	}

	id, err := h.service.CreatePayment(r.Context(), prID, adminID, in)
	switch {
	case err == nil:
		utils.Created(w, "Pembayaran berhasil dicatat", map[string]any{"payment_id": id})
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRNotApproved):
		utils.Error(w, http.StatusConflict, "Purchase request harus berstatus approved sebelum dapat dicatat pembayarannya")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal mencatat pembayaran")
	}
}

// POST /api/pr/payments/{paymentId}/confirm
func (h *PRPaymentHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	paymentID, ok := utils.ParseIDParam(w, chi.URLParam(r, "paymentId"))
	if !ok {
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	err := h.service.ConfirmPaid(r.Context(), paymentID, adminID)
	switch {
	case err == nil:
		utils.OK(w, "Pembayaran berhasil dikonfirmasi lunas", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Data pembayaran tidak ditemukan")
	case errors.Is(err, service.ErrPaymentMakerCheckerViolation):
		utils.Error(w, http.StatusForbidden, "Admin yang mencatat pembayaran tidak boleh merangkap mengonfirmasi pembayaran yang sama")
	case errors.Is(err, service.ErrPaymentAlreadyPaid):
		utils.Error(w, http.StatusConflict, "Pembayaran ini sudah dikonfirmasi lunas sebelumnya")
	case errors.Is(err, service.ErrPaymentAlreadyCancelled):
		utils.Error(w, http.StatusConflict, "Pembayaran ini sudah dibatalkan")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal mengonfirmasi pembayaran")
	}
}

// POST /api/pr/payments/{paymentId}/cancel
func (h *PRPaymentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	paymentID, ok := utils.ParseIDParam(w, chi.URLParam(r, "paymentId"))
	if !ok {
		return
	}
	var req dto.CancelPRPaymentRequest
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

	err := h.service.CancelPayment(r.Context(), paymentID, adminID, req.Notes)
	switch {
	case err == nil:
		utils.OK(w, "Pembayaran berhasil dibatalkan", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Data pembayaran tidak ditemukan")
	case errors.Is(err, service.ErrPaymentAlreadyPaid):
		utils.Error(w, http.StatusConflict, "Pembayaran yang sudah lunas tidak dapat dibatalkan lewat jalur ini")
	case errors.Is(err, service.ErrPaymentAlreadyCancelled):
		utils.Error(w, http.StatusConflict, "Pembayaran ini sudah dibatalkan sebelumnya")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal membatalkan pembayaran")
	}
}

func (h *PRPaymentHandler) PaymentChain(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	chain, err := h.service.GetPaymentChainHistory(r.Context(), prID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil riwayat pembayaran rantai PR")
		return
	}

	out := make([]map[string]any, 0, len(chain))
	for _, stage := range chain {
		out = append(out, map[string]any{
			"pr_id":     stage.PRID,
			"rfp_no":    stage.RfpNo,
			"pr_status": stage.Status,
			"payments":  dto.NewPRPaymentResponseList(stage.Payments),
		})
	}
	utils.OK(w, "Fetch success", out)
}

func (h *PRPaymentHandler) Export(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := utils.ParseDateParam(q.Get("start_date"))
	endDate := utils.ParseDateParam(q.Get("end_date"))
	if (startDate == "") != (endDate == "") {
		utils.Error(w, http.StatusBadRequest, "start_date dan end_date harus diisi bersamaan")
		return
	}

	status := strings.ToLower(strings.TrimSpace(q.Get("status")))
	if status != "" && !validPaymentExportStatus[status] {
		utils.Error(w, http.StatusBadRequest, "status tidak valid (gunakan: pending, paid, cancelled, draft)")
		return
	}

	filter := service.PRPaymentExportFilter{
		StartDate:     startDate,
		EndDate:       endDate,
		ResponsibleID: utils.AtoiDefault(q.Get("responsible"), 0),
		Status:        status,
		AdminID:       q.Get("user"),
	}

	data, filename, err := h.exportService.GenerateXlsx(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat file export summary payment")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Write(data)
}