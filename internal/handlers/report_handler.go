package handlers

import (
	"context"
	"net/http"
	"strconv"

	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type ReportHandler struct {
	poRepo *repository.PORepo
	logger *service.Logger
}

func NewReportHandler(poRepo *repository.PORepo, logger *service.Logger) *ReportHandler {
	return &ReportHandler{poRepo: poRepo, logger: logger}
}

// resolveYear membaca ?year= dari query string; kalau kosong/tidak valid,
// fallback ke tahun PO paling baru. Dipakai sebagai fallback SAJA - kalau
// caller mengirim start_date & end_date eksplisit, resolveYear tetap
// dipanggil untuk mengisi field "year" di response, tapi hasilnya tidak
// menentukan filter (start_date/end_date yang menang, lihat repository).
func (h *ReportHandler) resolveYear(ctx context.Context, raw string) (int, error) {
	if raw != "" {
		if y, err := strconv.Atoi(raw); err == nil && y > 0 {
			return y, nil
		}
	}
	return h.poRepo.LatestYear(ctx)
}

// GET /api/reports/years
func (h *ReportHandler) Years(w http.ResponseWriter, r *http.Request) {
	years, err := h.poRepo.ListYears(r.Context())
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "reports", Action: "years_failed", Status: service.LogStatusError,
			Message: err.Error(), IP: utils.ClientIP(r),
		})
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar tahun PO")
		return
	}
	utils.OK(w, "Fetch success", map[string]any{"years": years})
}

// GET /api/reports/chart?year=&start_date=&end_date=&region=&status=&acsg_pic=1
func (h *ReportHandler) Chart(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := utils.ParseDateParam(q.Get("start_date"))
	endDate := utils.ParseDateParam(q.Get("end_date"))

	var year int
	if startDate == "" || endDate == "" {
		y, err := h.resolveYear(r.Context(), q.Get("year"))
		if err != nil {
			h.logger.Log(service.LogEntry{
				Module: "reports", Action: "chart_resolve_year_failed", Status: service.LogStatusError,
				Message: err.Error(), IP: utils.ClientIP(r),
			})
			utils.Error(w, http.StatusInternalServerError, "Gagal menentukan tahun default")
			return
		}
		year = y
	} else if raw := q.Get("year"); raw != "" {
		if y, err := strconv.Atoi(raw); err == nil {
			year = y
		}
	}

	if (startDate == "") != (endDate == "") {
		utils.Error(w, http.StatusBadRequest, "start_date dan end_date harus diisi bersamaan")
		return
	}

	regionID := utils.AtoiDefault(q.Get("region"), 0)
	status := q.Get("status")
	picID := ""
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			picID = adminID
		}
	}

	data, err := h.poRepo.ReportChart(r.Context(), year, startDate, endDate, regionID, picID, status)
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "reports", Action: "chart_failed", Status: service.LogStatusError,
			Message: err.Error(), IP: utils.ClientIP(r),
			Metadata: map[string]any{"year": year, "start_date": startDate, "end_date": endDate, "region": regionID, "status": status},
		})
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data chart")
		return
	}
	utils.OK(w, "Fetch success", map[string]any{
		"year": year, "start_date": startDate, "end_date": endDate, "data": data,
	})
}

// GET /api/reports/bars?year=&start_date=&end_date=&status=
func (h *ReportHandler) Bars(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := utils.ParseDateParam(q.Get("start_date"))
	endDate := utils.ParseDateParam(q.Get("end_date"))

	year, err := h.resolveYear(r.Context(), q.Get("year"))
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "reports", Action: "bars_resolve_year_failed", Status: service.LogStatusError,
			Message: err.Error(), IP: utils.ClientIP(r),
		})
		utils.Error(w, http.StatusInternalServerError, "Gagal menentukan tahun default")
		return
	}
	status := q.Get("status")

	data, err := h.poRepo.ReportBarsByRegion(r.Context(), year, startDate, endDate, status)
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "reports", Action: "bars_failed", Status: service.LogStatusError,
			Message: err.Error(), IP: utils.ClientIP(r),
			Metadata: map[string]any{"year": year, "start_date": startDate, "end_date": endDate, "status": status},
		})
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data grafik regional")
		return
	}
	utils.OK(w, "Fetch success", map[string]any{
		"year": year, "start_date": startDate, "end_date": endDate, "data": data,
	})
}

// GET /api/reports/stats?acsg_pic=1
func (h *ReportHandler) Stats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	picID := ""
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			picID = adminID
		}
	}
	data, err := h.poRepo.StatByRegion(r.Context(), picID)
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "reports", Action: "stats_failed", Status: service.LogStatusError,
			Message: err.Error(), IP: utils.ClientIP(r),
		})
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil statistik region")
		return
	}
	utils.OK(w, "Fetch success", data)
}