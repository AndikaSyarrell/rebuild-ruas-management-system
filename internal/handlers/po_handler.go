package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type POHandler struct {
	repo          *repository.PORepo
	activityRepo  *repository.ActivityRepo
	clientRepo    *repository.ClientRepo
	ppnRepo       *repository.PpnRepo
	exportService *service.ExportService
	poService     *service.POService          // BARU - PR-PO Linking
	quotationRepo *repository.PRQuotationRepo // BARU - PR-PO Linking
}

func NewPOHandler(repo *repository.PORepo, activityRepo *repository.ActivityRepo, clientRepo *repository.ClientRepo, ppnRepo *repository.PpnRepo, exportService *service.ExportService, poService *service.POService, quotationRepo *repository.PRQuotationRepo) *POHandler {
	return &POHandler{repo: repo, activityRepo: activityRepo, clientRepo: clientRepo, ppnRepo: ppnRepo, exportService: exportService, poService: poService, quotationRepo: quotationRepo}
}

func (h *POHandler) logActivity(r *http.Request, poID, actType, notes string) {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		return
	}
	_, _ = h.activityRepo.Insert(r.Context(), poID, adminID, actType, notes)
}

// GET /api/po?status=&keyword=&region=&start_date=&end_date=&sortby=&page=&item=
func (h *POHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	q := r.URL.Query()

	status := q.Get("status")
	if status == "" {
		status = "open"
	}

	filter := repository.ListFilter{
		Page:      p.Page,
		PerPage:   p.PerPage,
		Status:    status,
		Keyword:   q.Get("keyword"),
		StartDate: utils.ParseDateParam(q.Get("start_date")),
		EndDate:   utils.ParseDateParam(q.Get("end_date")),
		RegionID:  utils.AtoiDefault(q.Get("region"), 0),
		SortBy:    q.Get("sortby"),
	}
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			filter.PicID = adminID
		}
	}

	data, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data PO")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewPOListItemResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

// GET /api/po/export?...
func (h *POHandler) Export(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.ExportFilter{
		StartDate:  utils.ParseDateParam(q.Get("start_date")),
		EndDate:    utils.ParseDateParam(q.Get("end_date")),
		DivisionID: utils.AtoiDefault(q.Get("division"), 0),
		Status:     q.Get("status"),
		RegionID:   utils.AtoiDefault(q.Get("region"), 0),
		ClientID:   utils.AtoiDefault(q.Get("client"), 0),
	}

	path, cacheHit, err := h.exportService.GetOrGenerate(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat file export")
		return
	}

	f, err := os.Open(path)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuka file export")
		return
	}
	defer f.Close()

	filename := fmt.Sprintf("PO_Export_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if cacheHit {
		w.Header().Set("X-Export-Cache", "HIT")
	} else {
		w.Header().Set("X-Export-Cache", "MISS")
	}

	io.Copy(w, f)
}

// GET /api/po/total?...
func (h *POHandler) Total(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.ListFilter{
		Keyword:   q.Get("keyword"),
		StartDate: utils.ParseDateParam(q.Get("start_date")),
		EndDate:   utils.ParseDateParam(q.Get("end_date")),
		RegionID:  utils.AtoiDefault(q.Get("region"), 0),
	}
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			filter.PicID = adminID
		}
	}

	counts, err := h.repo.CountByFilter(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil ringkasan PO")
		return
	}
	utils.OK(w, "Fetch success", counts)
}

// GET /api/po/{id}
func (h *POHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	po, err := h.repo.GetDetail(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}
	items, err := h.repo.ListItems(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil item PO")
		return
	}

	quotations, err := h.quotationRepo.ListByPO(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar quotation ter-link")
		return
	}
	linked := make([]dto.LinkedQuotationResponse, 0, len(quotations))
	for _, q := range quotations {
		members, err := h.quotationRepo.MemberPRs(r.Context(), q.ID)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal mengambil anggota quotation")
			return
		}
		linked = append(linked, dto.NewLinkedQuotationResponse(q, members))
	}

	utils.OK(w, "Fetch success", dto.NewPOWithItemsResponse(*po, items, linked))
}

// POST /api/po
func (h *POHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePORequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	date := utils.ParseDateParam(req.Date)
	if date == "" {
		utils.Error(w, http.StatusBadRequest, "Tanggal PO tidak valid")
		return
	}

	exists, err := h.repo.OrderNumExists(r.Context(), req.OrderNum)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa nomor PO")
		return
	}
	if exists {
		utils.Error(w, http.StatusConflict, "Nomor PO sudah terdaftar")
		return
	}

	clientID := int(req.ClientID)
	if clientID == 0 {
		if req.ClientEmail == "" || req.ClientName == "" {
			utils.Error(w, http.StatusBadRequest, "Data klien tidak lengkap")
			return
		}
		existingClient, err := h.clientRepo.GetByEmail(r.Context(), req.ClientEmail)
		if err == nil && existingClient != nil {
			if existingClient.Active != "yes" {
				utils.Error(w, http.StatusBadRequest, "Klien dengan email tersebut sudah terdaftar namun berstatus nonaktif")
				return
			}
			clientID = existingClient.ID
			req.ClientName = existingClient.Name
			req.ClientPhone = existingClient.Phone
			req.ClientAddr = existingClient.Address
		} else {
			newID, err := h.clientRepo.Create(r.Context(), req.ClientName, req.ClientPhone, req.ClientEmail, req.ClientAddr, 0)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, "Gagal membuat data klien baru")
				return
			}
			clientID = int(newID)
		}
	} else {
		existingClient, err := h.clientRepo.GetByID(r.Context(), clientID)
		if err != nil {
			utils.Error(w, http.StatusBadRequest,
				fmt.Sprintf("Client dengan id %d tidak ditemukan, isi data client secara lengkap (client_name, client_email, client_phone, client_address) untuk membuat client baru", clientID))
			return
		}
		if existingClient.Active != "yes" {
			utils.Error(w, http.StatusBadRequest, "Klien ini berstatus nonaktif, tidak dapat dipakai untuk membuat PO baru")
			return
		}
		req.ClientName = existingClient.Name
		req.ClientEmail = existingClient.Email
		req.ClientPhone = existingClient.Phone
		req.ClientAddr = existingClient.Address
	}

	ppn, err := h.ppnRepo.GetByID(r.Context(), int(req.PpnID))
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "Tarif PPN tidak valid")
		return
	}

	adminID, _ := actorFromContext(r.Context())
	poID := utils.GenerateSequentialID("PO")

	items := make([]repository.NewPOItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, repository.NewPOItemInput{
			Desc: it.Desc, Product: it.Product, Qty: int(it.Qty), UnitID: int(it.UnitID), Price: float64(it.Price),
		})
	}

	in := repository.NewPOInput{
		ID: poID, OrderNum: req.OrderNum, RegionID: int(req.RegionID), AdminID: adminID, PicID: req.PicID,
		PicClientID: req.PicClientID,
		DivisionID: req.DivisionIDPtr(), PpnID: int(req.PpnID), PpnRate: ppn.Value, Date: date,
		ClientID: clientID, ClientName: req.ClientName, ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone, ClientAddr: req.ClientAddr, SubClient: req.SubClient, Items: items,
	}

	if err := h.repo.Create(r.Context(), in); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat PO")
		return
	}

	if hint := repository.NormalizeQuotationNo(req.QuotNo); hint != "" {
		_ = h.repo.UpdateQuotHint(r.Context(), poID, hint)
	}

	h.logActivity(r, poID, "open", "Purchase order baru dibuat")
	utils.Created(w, "PO berhasil dibuat", map[string]any{"po_id": poID})
}

func (h *POHandler) UpdateQuotHint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateQuotHintRequest
	if err := decodeJSON(r, &req); err != nil{
		utils.Error(w, http.StatusBadRequest, "Body request tidak valid")
		return
	}
	req.Normalize()
	if _, err := h.repo.GetDetail(r.Context(), id); err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}
	if err := h.repo.UpdateQuotHint(r.Context(), id, req.QuotNo); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbaru hint qoutation")
		return
	}
	h.logActivity(r, id, "edit", "memperbarui hint nomor qoutation")
	utils.OK(w, "hint quotation berhasil diperbarui", nil)
}

// PUT /api/po/{id}
func (h *POHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdatePORequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	date := utils.ParseDateParam(req.Date)

	current, err := h.repo.GetDetail(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}
	if current.OrderNum != req.OrderNum {
		exists, err := h.repo.OrderNumExists(r.Context(), req.OrderNum)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa nomor PO")
			return
		}
		if exists {
			utils.Error(w, http.StatusConflict, "Nomor PO sudah terdaftar")
			return
		}
	}

	clientID := int(req.ClientID)
	clientName, clientEmail := req.ClientName, req.ClientEmail
	clientPhone, clientAddr := req.ClientPhone, req.ClientAddr

	if clientID != 0 {
		existingClient, err := h.clientRepo.GetByID(r.Context(), clientID)
		if err != nil {
			utils.Error(w, http.StatusBadRequest,
				fmt.Sprintf("Client dengan id %d tidak ditemukan", clientID))
			return
		}
		if existingClient.Active != "yes" {
			utils.Error(w, http.StatusBadRequest, "Klien ini berstatus nonaktif, tidak dapat dipakai untuk update PO baru")
			return
		}
		clientName = existingClient.Name
		clientEmail = existingClient.Email
		clientPhone = existingClient.Phone
		clientAddr = existingClient.Address
	} else if clientName == "" || clientEmail == "" {
		utils.Error(w, http.StatusBadRequest, "Data klien tidak lengkap")
		return
	}

	err = h.repo.UpdateHeader(r.Context(), id, repository.UpdatePOInput{
		OrderNum: req.OrderNum, RegionID: int(req.RegionID), PicID: req.PicID, PicClientID: req.PicClientID,
		DivisionID: req.DivisionIDPtr(), Date: date,
		ClientID: clientID, ClientName: clientName, ClientEmail: clientEmail,
		ClientPhone: clientPhone, ClientAddr: clientAddr, SubClient: req.SubClient,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui PO")
		return
	}

	h.logActivity(r, id, "edit", "Mengubah detail informasi purchase order")
	utils.OK(w, "PO berhasil diperbarui", nil)
}

// POST /api/po/{id}/status
func (h *POHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.ChangeStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.ChangeStatus(r.Context(), id, req.Status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengubah status PO")
		return
	}
	if req.Status == "cancel" || req.Status == "open" {
		linked, err := h.quotationRepo.ListByPO(r.Context(), id)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa link quotation")
			return
		}
		if len(linked) > 0 {
			msg := "PO masih ter-link ke quotation, cabut link terlebih dahulu sebelum membatalkan PO"
			if req.Status == "open" {
				msg = "PO masih ter-link ke quotation, cabut link terlebih dahulu sebelum mengembalikan status PO ke open"
			}
			utils.Error(w, http.StatusConflict, msg)
			return
		}
	}
	if err := h.repo.ChangeStatus(r.Context(), id, req.Status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengubah status PO")
		return
	}
	h.logActivity(r, id, req.Status, "Mengubah status menjadi "+req.Status)
	utils.OK(w, "Status PO berhasil diperbarui", nil)
}

// POST /api/po/{id}/paid
func (h *POHandler) ChangePaid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.ChangePaidRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.ChangePaid(r.Context(), id, req.Paid); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengubah status pembayaran")
		return
	}
	actType, notes := "notpaid", "Set purchase order ke belum dibayar"
	if req.Paid == "yes" {
		actType, notes = "paid", "Set purchase order sudah dibayar"
	}
	h.logActivity(r, id, actType, notes)
	utils.OK(w, "Status pembayaran berhasil diperbarui", nil)
}

// POST /api/po/{id}/invoice
func (h *POHandler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateInvoiceRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.UpdateInvoice(r.Context(), id, req.Invoice); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui invoice")
		return
	}
	h.logActivity(r, id, "invoice", "Memperbarui nomor invoice")
	utils.OK(w, "Invoice berhasil diperbarui", nil)
}

// POST /api/po/{id}/notes
func (h *POHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateNotesRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := h.repo.UpdateNotes(r.Context(), id, req.Notes); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui catatan PO")
		return
	}
	h.logActivity(r, id, "notes", "Memperbarui catatan purchase order")
	utils.OK(w, "Catatan PO berhasil diperbarui", nil)
}

// DELETE /api/po/{id}
func (h *POHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID, _ := actorFromContext(r.Context())
	err := h.poService.Delete(r.Context(), id, adminID)
	switch {
	case err == nil:
		utils.OK(w, "PO berhasil dihapus", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan.")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal Menghapus PO")
	}
}

func (h *POHandler) QuotationCandidates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	keyword := r.URL.Query().Get("keyword")

	candidates, err := h.poService.ListQuotationCandidates(r.Context(), id, keyword)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar kandidat quotation")
		return
	}
	utils.OK(w, "Fetch success", dto.NewQuotationCandidateResponseList(candidates))
}

func (h *POHandler) LinkQuotation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.LinkQuotationRequest
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

	err := h.poService.LinkQuotation(r.Context(), id, int(req.QuotationID), adminID)
	switch {
	case err == nil:
		h.logActivity(r, id, "link_quotation", "Menautkan quotation ke purchase order ini")
		utils.OK(w, "Quotation berhasil di-link ke PO", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "PO atau quotation tidak ditemukan")
	case errors.Is(err, service.ErrPONotLinkable):
		utils.Error(w, http.StatusConflict, "PO harus berstatus prepared, progress, atau complete sebelum dapat di-link ke quotation")
	case errors.Is(err, service.ErrQuotationAlreadyLinked):
		utils.Error(w, http.StatusConflict, "Quotation ini sudah ter-link ke PO lain")
	case errors.Is(err, service.ErrQuotationNotEligible):
		utils.Error(w, http.StatusConflict, "Belum ada purchase request anggota grup quotation ini yang berstatus completed")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal menautkan quotation ke PO")
	}
}

// POST /api/po/{id}/unlink-quotation
func (h *POHandler) UnlinkQuotation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UnlinkQuotationRequest
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

	err := h.poService.UnlinkQuotation(r.Context(), id, int(req.QuotationID), adminID, req.Notes)
	switch {
	case err == nil:
		h.logActivity(r, id, "unlink_quotation", "Mencabut link quotation dari purchase order ini: "+req.Notes)
		utils.OK(w, "Link quotation berhasil dicabut", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Quotation tidak ditemukan")
	case errors.Is(err, service.ErrQuotationNotLinkedToThisPO):
		utils.Error(w, http.StatusBadRequest, "Quotation ini tidak ter-link ke PO yang dimaksud")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal mencabut link quotation")
	}
}

// --- Items ---

// GET /api/po/{id}/items
func (h *POHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	items, err := h.repo.ListItems(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil item PO")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPOItemResponseList(items))
}

// POST /api/po/{id}/items
func (h *POHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.POItemRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	po, err := h.repo.GetDetail(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}

	if _, err := h.repo.AddItem(r.Context(), id, int(req.UnitID), req.Product, req.Desc, int(req.Qty), float64(req.Price)); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menambah item")
		return
	}
	if err := h.repo.RecomputeTotals(r.Context(), id, po.PpnRate); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Item ditambahkan, namun gagal menghitung ulang total")
		return
	}

	h.logActivity(r, id, "add", "Menambah item baru pada daftar")
	utils.Created(w, "Item berhasil ditambahkan", nil)
}

// PUT /api/po/items/{itemId}
func (h *POHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := utils.ParseIDParam(w, chi.URLParam(r, "itemId"))
	if !ok {
		return
	}
	var req dto.POItemRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	poID, err := h.repo.GetItemPO(r.Context(), itemID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Item tidak ditemukan")
		return
	}
	po, err := h.repo.GetDetail(r.Context(), poID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}

	if err := h.repo.UpdateItem(r.Context(), itemID, int(req.UnitID), req.Product, req.Desc, int(req.Qty), float64(req.Price)); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui item")
		return
	}
	if err := h.repo.RecomputeTotals(r.Context(), poID, po.PpnRate); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Item diperbarui, namun gagal menghitung ulang total")
		return
	}

	h.logActivity(r, poID, "edit", "Mengubah item pada daftar")
	utils.OK(w, "Item berhasil diperbarui", nil)
}

// DELETE /api/po/items/{itemId}
func (h *POHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := utils.ParseIDParam(w, chi.URLParam(r, "itemId"))
	if !ok {
		return
	}

	poID, err := h.repo.GetItemPO(r.Context(), itemID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Item tidak ditemukan")
		return
	}
	po, err := h.repo.GetDetail(r.Context(), poID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}

	if err := h.repo.DeleteItem(r.Context(), itemID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus item")
		return
	}
	if err := h.repo.RecomputeTotals(r.Context(), poID, po.PpnRate); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Item dihapus, namun gagal menghitung ulang total")
		return
	}

	h.logActivity(r, poID, "delete", "Menghapus item pada daftar")
	utils.OK(w, "Item berhasil dihapus", nil)
}