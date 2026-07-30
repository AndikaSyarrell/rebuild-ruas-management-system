package dto

import (
	"strings"
	"time"

	"rms-backend/internal/models"
)

// POItemResponse adalah representasi 1 item PO yang aman & ringkas untuk
// dikirim ke client (dipisah dari models.POItem supaya perubahan skema DB
// tidak otomatis bocor ke response API).
type POItemResponse struct {
	ItemID    int     `json:"item_id"`
	Product   string  `json:"item_product"`
	Desc      string  `json:"item_desc,omitempty"`
	Qty       int     `json:"item_qty"`
	Price     float64 `json:"item_price"`
	UnitTitle string  `json:"unit_title,omitempty"`
}

func NewPOItemResponse(m models.POItem) POItemResponse {
	desc := ""
	if m.Desc != nil {
		desc = *m.Desc
	}
	return POItemResponse{
		ItemID:    m.ID,
		Product:   m.Product,
		Desc:      desc,
		Qty:       m.Qty,
		Price:     m.Price,
		UnitTitle: m.UnitTitle,
	}
}

func NewPOItemResponseList(items []models.POItem) []POItemResponse {
	out := make([]POItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, NewPOItemResponse(it))
	}
	return out
}

// POResponse adalah representasi detail lengkap 1 PO untuk endpoint Detail.
type POResponse struct {
	ID        string `json:"po_id"`
	OrderNum  string `json:"po_order_num"`
	Invoice   string `json:"po_invoice,omitempty"`
	Status    string `json:"po_status"`
	Document  string `json:"po_document"`
	Paid      string `json:"po_paid"`
	Notes     string `json:"po_notes,omitempty"`
	SubClient string `json:"po_subclient,omitempty"`

	Subtotal  float64 `json:"po_subtotal"`
	PpnRate   float64 `json:"po_ppn_rate"`
	PpnAmount float64 `json:"po_ppn_amount"`
	Total     float64 `json:"po_total"`
	ItemTotal int     `json:"po_item_total"`

	ClientName  string `json:"client_name"`
	ClientEmail string `json:"client_email"`
	ClientPhone string `json:"client_phone"`
	ClientAddr  string `json:"client_address"`

	AdminName    string `json:"admin_name,omitempty"`
	PicName      string `json:"pic_name,omitempty"`
	RegionTitle  string `json:"region_title,omitempty"`
	DivisionName string `json:"division_title,omitempty"`
	ProductNames []string `json:"product_names"`

	Date         string `json:"po_date,omitempty"`
	ExpDate      string `json:"po_exp_date,omitempty"`
	PreparedDate string `json:"po_prepared_date,omitempty"`
	ProgressDate string `json:"po_progress_date,omitempty"`
	CompleteDate string `json:"po_complete_date,omitempty"`
	CancelDate   string `json:"po_cancel_date,omitempty"`
	CreateDate   string `json:"po_create_date"`
	ModifyDate   string `json:"po_modify_date"`
}

func NewPOResponse(m models.PO) POResponse {
	invoice := ""
	if m.Invoice != nil {
		invoice = *m.Invoice
	}
	notes := ""
	if m.Notes != nil {
		notes = *m.Notes
	}
	subClient := ""
	if m.SubClient != nil {
		subClient = *m.SubClient
	}

	return POResponse{
		ID:        m.ID,
		OrderNum:  m.OrderNum,
		Invoice:   invoice,
		Status:    m.Status,
		Document:  m.Document,
		Paid:      m.Paid,
		Notes:     notes,
		SubClient: subClient,

		Subtotal:  m.Subtotal,
		PpnRate:   m.PpnRate,
		PpnAmount: m.PpnAmount,
		Total:     m.Total,
		ItemTotal: m.ItemTotal,

		ClientName:  m.ClientName,
		ClientEmail: m.ClientEmail,
		ClientPhone: m.ClientPhone,
		ClientAddr:  m.ClientAddr,

		AdminName:    m.AdminName,
		PicName:      m.PicName,
		RegionTitle:  m.RegionTitle,
		DivisionName: m.DivisionName,

		Date:         formatDate(m.Date),
		ExpDate:      formatDate(m.ExpDate),
		PreparedDate: formatDateTime(m.PreparedDate),
		ProgressDate: formatDateTime(m.ProgressDate),
		CompleteDate: formatDateTime(m.CompleteDate),
		CancelDate:   formatDateTime(m.CancelDate),
		CreateDate:   m.CreateDate.Format(time.RFC3339),
		ModifyDate:   m.ModifyDate.Format(time.RFC3339),
	}
}

// POWithItemsResponse membungkus detail PO + daftar itemnya, dipakai oleh
// endpoint GET /po/{id}.
type POWithItemsResponse struct {
	PO    POResponse       `json:"po"`
	Items []POItemResponse `json:"items"`
}

func NewPOWithItemsResponse(po models.PO, items []models.POItem) POWithItemsResponse {
	return POWithItemsResponse{
		PO:    NewPOResponse(po),
		Items: NewPOItemResponseList(items),
	}
}

// POListItemResponse adalah representasi ringkas 1 baris PO untuk endpoint
// List/dashboard - sengaja tidak menyertakan field yang tidak diisi oleh
// query list (mis. Subtotal/PpnAmount/AdminName) supaya tidak menampilkan
// nilai kosong yang menyesatkan seolah memang bernilai 0.
type POListItemResponse struct {
	ID           string  `json:"po_id"`
	OrderNum     string  `json:"po_order_num"`
	Invoice      string  `json:"po_invoice,omitempty"`
	Status       string  `json:"po_status"`
	Total        float64 `json:"po_total"`
	Document     string  `json:"po_document,omitempty"`
	Paid         string  `json:"po_paid,omitempty"`
	SubClient    string  `json:"po_subclient,omitempty"`
	Date         string  `json:"po_date,omitempty"`
	ExpDate      string  `json:"po_exp_date,omitempty"`
	ClientName   string  `json:"client_name,omitempty"`
	PicName      string  `json:"pic_name,omitempty"`
	DivisionName string  `json:"division_title,omitempty"`
	RegionTitle  string  `json:"region_title,omitempty"`
	ProductNames []string `json:"product_names"`
}

func splitProductNames(raw string) []string {
	out := []string{}
	if raw == "" {
		return out
	}
	for _, p := range strings.Split(raw, "||") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func NewPOListItemResponse(m models.PO) POListItemResponse {
	invoice := ""
	if m.Invoice != nil {
		invoice = *m.Invoice
	}
	subClient := ""
	if m.SubClient != nil {
		subClient = *m.SubClient
	}
	return POListItemResponse{
		ID:           m.ID,
		OrderNum:     m.OrderNum,
		Invoice:      invoice,
		Status:       m.Status,
		Total:        m.Total,
		Document:     m.Document,
		Paid:         m.Paid,
		SubClient:    subClient,
		Date:         formatDate(m.Date),
		ExpDate:      formatDate(m.ExpDate),
		ClientName:   m.ClientName,
		PicName:      m.PicName,
		DivisionName: m.DivisionName,
		RegionTitle:  m.RegionTitle,
		ProductNames: splitProductNames(m.ProductNames),
	}
}

func NewPOListItemResponseList(list []models.PO) []POListItemResponse {
	out := make([]POListItemResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPOListItemResponse(m))
	}
	return out
}

// --- helpers ---

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatDateTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}