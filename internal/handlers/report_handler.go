package handlers

import (
	"net/http"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type ReportHandler struct {
	poRepo *repository.PORepo
}

func NewReportHandler(poRepo *repository.PORepo) *ReportHandler {
	return &ReportHandler{poRepo: poRepo}
}

// GET /api/reports/chart?start_date=&end_date=&region=&status=&acsg_pic=1
func (h *ReportHandler) Chart(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start := utils.ParseDateParam(q.Get("start_date"))
	end := utils.ParseDateParam(q.Get("end_date"))
	regionID := utils.AtoiDefault(q.Get("region"), 0)
	status := q.Get("status")
	picID := ""
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			picID = adminID
		}
	}

	data, err := h.poRepo.ReportChart(r.Context(), start, end, regionID, picID, status)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data chart")
		return
	}
	utils.OK(w, "Fetch success", data)
}

// GET /api/reports/bars?start_date=&end_date=&status=
func (h *ReportHandler) Bars(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start := utils.ParseDateParam(q.Get("start_date"))
	end := utils.ParseDateParam(q.Get("end_date"))
	status := q.Get("status")

	data, err := h.poRepo.ReportBarsByRegion(r.Context(), start, end, status)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data grafik regional")
		return
	}
	utils.OK(w, "Fetch success", data)
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
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil statistik region")
		return
	}
	utils.OK(w, "Fetch success", data)
}
