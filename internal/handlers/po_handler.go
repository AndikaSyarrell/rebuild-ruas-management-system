package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	"log"

	"github.com/go-chi/chi/v5"

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
}

func NewPOHandler(repo *repository.PORepo, activityRepo *repository.ActivityRepo, clientRepo *repository.ClientRepo, ppnRepo *repository.PpnRepo, exportService *service.ExportService) *POHandler {
	return &POHandler{repo: repo, activityRepo: activityRepo, clientRepo: clientRepo, ppnRepo: ppnRepo, exportService: exportService}
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
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", data, map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

// GET /api/po/export?start_date=&end_date=&division=&status=&region=&client=
//
// Export breakdown per-item (1 baris = 1 item PO) ke .xlsx. Hasil di-cache di
// disk selama TTL (lihat PO_EXPORT_TTL) berdasarkan kombinasi filter -
// permintaan berikutnya dengan filter identik dalam TTL yang sama langsung
// disajikan dari file cache tanpa query ulang ke database.
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
    log.Printf("export PO failed: %v", err) // TEMP: see internal/handlers/po_handler.go imports, add "log"
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

// GET /api/po/total?...  - ringkasan jumlah per status untuk badge
func (h *POHandler) Total(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	regionID := utils.AtoiDefault(q.Get("region"), 0)
	picID := ""
	if q.Get("acsg_pic") == "1" {
		if adminID, ok := actorFromContext(r.Context()); ok {
			picID = adminID
		}
	}
	counts, err := h.repo.CountByStatus(r.Context(), regionID, picID)
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
	utils.OK(w, "Fetch success", map[string]any{"po": po, "items": items})
}

type createPOItemRequest struct {
	Desc    string  `json:"desc"`
	Product string  `json:"product"`
	Qty     int     `json:"qty"`
	UnitID  int     `json:"unit_id"`
	Price   float64 `json:"price"`
}

type createPORequest struct {
	OrderNum    string                `json:"order_num"`
	RegionID    int                   `json:"region_id"`
	PicID       string                `json:"pic_id"`
	DivisionID  *int                  `json:"division_id"`
	PpnID       int                   `json:"ppn_id"`
	Date        string                `json:"date"`
	ClientID    int                   `json:"client_id"`
	ClientName  string                `json:"client_name"`
	ClientEmail string                `json:"client_email"`
	ClientPhone string                `json:"client_phone"`
	ClientAddr  string                `json:"client_address"`
	SubClient   string                `json:"sub_client"`
	Items       []createPOItemRequest `json:"items"`
}

// POST /api/po
func (h *POHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPORequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if req.OrderNum == "" || req.RegionID == 0 || req.PicID == "" || len(req.Items) == 0 {
		utils.Error(w, http.StatusBadRequest, "Nomor PO, region, PIC, dan minimal 1 item wajib diisi")
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

	// klien baru (belum punya client_id) langsung dibuat sebagai T_Client baru.
	clientID := req.ClientID
	if clientID == 0 {
		// Tidak ada client_id -> anggap klien baru, wajib isi data lengkap manual.
		if req.ClientEmail == "" || req.ClientName == "" {
			utils.Error(w, http.StatusBadRequest, "Data klien tidak lengkap")
			return
		}
		existingClient, err := h.clientRepo.GetByEmail(r.Context(), req.ClientEmail)
		if err == nil && existingClient != nil {
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
		// client_id dikirim -> ambil otomatis nama/email/telepon/alamat dari T_Client,
		// tidak perlu (dan tidak dipakai) meskipun field-field itu ikut dikirim di payload.
		existingClient, err := h.clientRepo.GetByID(r.Context(), clientID)
		if err != nil {
			utils.Error(w, http.StatusBadRequest,
				fmt.Sprintf("Client dengan id %d tidak ditemukan, isi data client secara lengkap (client_name, client_email, client_phone, client_address) untuk membuat client baru", clientID))
			return
		}
		req.ClientName = existingClient.Name
		req.ClientEmail = existingClient.Email
		req.ClientPhone = existingClient.Phone
		req.ClientAddr = existingClient.Address
	}

	ppn, err := h.ppnRepo.GetByID(r.Context(), req.PpnID)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "Tarif PPN tidak valid")
		return
	}

	adminID, _ := actorFromContext(r.Context())
	poID := utils.GenerateSequentialID("PO")

	items := make([]repository.NewPOItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Product == "" || it.Qty <= 0 || it.UnitID == 0 {
			utils.Error(w, http.StatusBadRequest, "Setiap item wajib memiliki produk, qty, dan satuan yang valid")
			return
		}
		items = append(items, repository.NewPOItemInput{
			Desc: it.Desc, Product: it.Product, Qty: it.Qty, UnitID: it.UnitID, Price: it.Price,
		})
	}

	in := repository.NewPOInput{
		ID: poID, OrderNum: req.OrderNum, RegionID: req.RegionID, AdminID: adminID, PicID: req.PicID,
		DivisionID: req.DivisionID, PpnID: req.PpnID, PpnRate: ppn.Value, Date: date,
		ClientID: clientID, ClientName: req.ClientName, ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone, ClientAddr: req.ClientAddr, SubClient: req.SubClient, Items: items,
	}

	if err := h.repo.Create(r.Context(), in); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat PO")
		return
	}

	h.logActivity(r, poID, "open", "Purchase order baru dibuat")

	utils.Created(w, "PO berhasil dibuat", map[string]any{"po_id": poID})
}

type updatePORequest struct {
	OrderNum    string `json:"order_num"`
	RegionID    int    `json:"region_id"`
	PicID       string `json:"pic_id"`
	DivisionID  *int   `json:"division_id"`
	Date        string `json:"date"`
	ClientID    int    `json:"client_id"`
	ClientName  string `json:"client_name"`
	ClientEmail string `json:"client_email"`
	ClientPhone string `json:"client_phone"`
	ClientAddr  string `json:"client_address"`
	SubClient   string `json:"sub_client"`
}

// PUT /api/po/{id}
func (h *POHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updatePORequest
	if err := decodeJSON(r, &req); err != nil || req.OrderNum == "" {
		utils.Error(w, http.StatusBadRequest, "Data PO tidak lengkap")
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

	err = h.repo.UpdateHeader(r.Context(), id, repository.UpdatePOInput{
		OrderNum: req.OrderNum, RegionID: req.RegionID, PicID: req.PicID, DivisionID: req.DivisionID,
		Date: date, ClientID: req.ClientID, ClientName: req.ClientName, ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone, ClientAddr: req.ClientAddr, SubClient: req.SubClient,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui PO")
		return
	}

	h.logActivity(r, id, "edit", "Mengubah detail informasi purchase order")
	utils.OK(w, "PO berhasil diperbarui", nil)
}

// POST /api/po/{id}/status  {"status": "progress"}
func (h *POHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Status string `json:"status"`
	}
	valid := map[string]bool{"open": true, "progress": true, "prepared": true, "complete": true, "cancel": true}
	if err := decodeJSON(r, &req); err != nil || !valid[req.Status] {
		utils.Error(w, http.StatusBadRequest, "Status tidak valid")
		return
	}
	if err := h.repo.ChangeStatus(r.Context(), id, req.Status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengubah status PO")
		return
	}
	h.logActivity(r, id, req.Status, "Mengubah status menjadi "+req.Status)
	utils.OK(w, "Status PO berhasil diperbarui", nil)
}

// POST /api/po/{id}/paid  {"paid": "yes"}
func (h *POHandler) ChangePaid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Paid string `json:"paid"`
	}
	if err := decodeJSON(r, &req); err != nil || (req.Paid != "yes" && req.Paid != "no") {
		utils.Error(w, http.StatusBadRequest, "Status pembayaran tidak valid")
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

// POST /api/po/{id}/invoice  {"invoice": "..."}
func (h *POHandler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Invoice string `json:"invoice"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Invoice == "" {
		utils.Error(w, http.StatusBadRequest, "Nomor invoice wajib diisi")
		return
	}
	if err := h.repo.UpdateInvoice(r.Context(), id, req.Invoice); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui invoice")
		return
	}
	h.logActivity(r, id, "invoice", "Memperbarui nomor invoice")
	utils.OK(w, "Invoice berhasil diperbarui", nil)
}

// POST /api/po/{id}/notes  {"notes": "..."}
func (h *POHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Notes string `json:"notes"`
	}
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
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus PO")
		return
	}
	utils.OK(w, "PO berhasil dihapus", nil)
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
	utils.OK(w, "Fetch success", items)
}

type itemRequest struct {
	Desc    string  `json:"desc"`
	Product string  `json:"product"`
	Qty     int     `json:"qty"`
	UnitID  int     `json:"unit_id"`
	Price   float64 `json:"price"`
}

// POST /api/po/{id}/items
func (h *POHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req itemRequest
	if err := decodeJSON(r, &req); err != nil || req.Product == "" || req.Qty <= 0 || req.UnitID == 0 {
		utils.Error(w, http.StatusBadRequest, "Data item tidak lengkap")
		return
	}

	po, err := h.repo.GetDetail(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}

	if _, err := h.repo.AddItem(r.Context(), id, req.UnitID, req.Product, req.Desc, req.Qty, req.Price); err != nil {
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
	itemID, _ := strconv.Atoi(chi.URLParam(r, "itemId"))
	var req itemRequest
	if err := decodeJSON(r, &req); err != nil || req.Product == "" || req.Qty <= 0 || req.UnitID == 0 {
		utils.Error(w, http.StatusBadRequest, "Data item tidak lengkap")
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

	if err := h.repo.UpdateItem(r.Context(), itemID, req.UnitID, req.Product, req.Desc, req.Qty, req.Price); err != nil {
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
	itemID, _ := strconv.Atoi(chi.URLParam(r, "itemId"))

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
