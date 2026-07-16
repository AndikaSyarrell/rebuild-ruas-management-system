package handlers

import (
	"net/http"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type DashboardHandler struct {
	poRepo *repository.PORepo
}

func NewDashboardHandler(poRepo *repository.PORepo) *DashboardHandler {
	return &DashboardHandler{poRepo: poRepo}
}

// GET /api/dashboard?region=&acsg_pic=1
func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	regionID := utils.AtoiDefault(q.Get("region"), 0)
	picID := ""
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			picID = adminID
		}
	}

	counts, err := h.poRepo.CountByStatus(r.Context(), regionID, picID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil ringkasan dashboard")
		return
	}

	recent, err := h.poRepo.ListDashboardOpen(r.Context(), regionID, picID, 10)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil PO terbaru")
		return
	}

	utils.OK(w, "Fetch success", map[string]any{
		"total_open":     counts["open"],
		"total_progress": counts["progress"],
		"total_prepared": counts["prepared"],
		"total_complete": counts["complete"],
		"total_cancel":   counts["cancel"],
		"recent_open_po": recent,
	})
}
