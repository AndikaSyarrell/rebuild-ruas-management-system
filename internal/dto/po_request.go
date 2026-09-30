package dto

import (
	"errors"

	"rms-backend/internal/utils"
	"rms-backend/internal/repository"
)

// CreatePOItemRequest adalah payload 1 baris item saat membuat PO baru.
type CreatePOItemRequest struct {
	Desc    string          `json:"desc"`
	Product string          `json:"product"`
	Qty     utils.FlexInt   `json:"qty"`
	UnitID  utils.FlexInt   `json:"unit_id"`
	Price   utils.FlexFloat `json:"price"`
}

func (r CreatePOItemRequest) Validate() error {
	if r.Product == "" {
		return errors.New("setiap item wajib memiliki nama produk")
	}
	if int(r.Qty) <= 0 {
		return errors.New("qty setiap item harus lebih dari 0")
	}
	if int(r.UnitID) == 0 {
		return errors.New("satuan setiap item wajib dipilih")
	}
	if float64(r.Price) < 0 {
		return errors.New("harga item tidak boleh negatif")
	}
	return nil
}

// CreatePORequest adalah payload lengkap pembuatan PO baru.
type CreatePORequest struct {
	OrderNum    string                 `json:"order_num"`
	RegionID    utils.FlexInt          `json:"region_id"`
	PicID       string                 `json:"pic_id"`
	PicClientID string                 `json:"pic_client_id"` // opsional
	DivisionID  *utils.FlexInt         `json:"division_id"`
	PpnID       utils.FlexInt          `json:"ppn_id"`
	Date        string                 `json:"date"`
	ClientID    utils.FlexInt          `json:"client_id"`
	ClientName  string                 `json:"client_name"`
	ClientEmail string                 `json:"client_email"`
	ClientPhone string                 `json:"client_phone"`
	ClientAddr  string                 `json:"client_address"`
	SubClient   string                 `json:"sub_client"`
	Items       []CreatePOItemRequest  `json:"items"`
	QuotNo string `json:"quot_no"`
}

type UpdateQuotHintRequest struct {
	QuotNo string `json:"quot_no"` // kosong = hapus hint
}

func (r *UpdateQuotHintRequest) Normalize() {
	r.QuotNo = repository.NormalizeQuotationNo(r.QuotNo)
}

func (r CreatePORequest) Validate() error {
	if r.OrderNum == "" {
		return errors.New("nomor PO wajib diisi")
	}
	if int(r.RegionID) == 0 {
		return errors.New("region wajib dipilih")
	}
	if r.PicID == "" {
		return errors.New("PIC wajib dipilih")
	}
	if len(r.Items) == 0 {
		return errors.New("PO wajib memiliki minimal 1 item")
	}
	for i, it := range r.Items {
		if err := it.Validate(); err != nil {
			return errors.New("item ke-" + itoa(i+1) + ": " + err.Error())
		}
	}
	return nil
}

// DivisionIDPtr mengonversi *FlexInt (bisa nil) ke *int biasa, siap dipakai
// oleh layer repository yang mengharapkan tipe native.
func (r CreatePORequest) DivisionIDPtr() *int {
	if r.DivisionID == nil {
		return nil
	}
	v := int(*r.DivisionID)
	return &v
}

// UpdatePORequest adalah payload update header PO (tidak termasuk item -
// item dikelola lewat endpoint /items terpisah).
type UpdatePORequest struct {
	OrderNum    string         `json:"order_num"`
	RegionID    utils.FlexInt  `json:"region_id"`
	PicID       string         `json:"pic_id"`
	PicClientID string         `json:"pic_client_id"` // opsional; string kosong = hapus PIC client
	DivisionID  *utils.FlexInt `json:"division_id"`
	Date        string         `json:"date"`
	ClientID    utils.FlexInt  `json:"client_id"`
	ClientName  string         `json:"client_name"`
	ClientEmail string         `json:"client_email"`
	ClientPhone string         `json:"client_phone"`
	ClientAddr  string         `json:"client_address"`
	SubClient   string         `json:"sub_client"`
}

func (r UpdatePORequest) Validate() error {
	if r.OrderNum == "" {
		return errors.New("nomor PO wajib diisi")
	}
	return nil
}

func (r UpdatePORequest) DivisionIDPtr() *int {
	if r.DivisionID == nil {
		return nil
	}
	v := int(*r.DivisionID)
	return &v
}

// AddPOItemRequest / UpdatePOItemRequest - payload tambah/ubah 1 item PO
// (endpoint terpisah dari create PO).
type POItemRequest struct {
	Desc    string          `json:"desc"`
	Product string          `json:"product"`
	Qty     utils.FlexInt   `json:"qty"`
	UnitID  utils.FlexInt   `json:"unit_id"`
	Price   utils.FlexFloat `json:"price"`
}

func (r POItemRequest) Validate() error {
	if r.Product == "" {
		return errors.New("produk wajib diisi")
	}
	if int(r.Qty) <= 0 {
		return errors.New("qty harus lebih dari 0")
	}
	if int(r.UnitID) == 0 {
		return errors.New("satuan wajib dipilih")
	}
	if float64(r.Price) < 0 {
		return errors.New("harga tidak boleh negatif")
	}
	return nil
}

// ChangeStatusRequest - payload ubah status PO.
type ChangeStatusRequest struct {
	Status string `json:"status"`
}

var validPOStatus = map[string]bool{
	"open": true, "progress": true, "prepared": true, "complete": true, "cancel": true,
}

func (r ChangeStatusRequest) Validate() error {
	if !validPOStatus[r.Status] {
		return errors.New("status tidak valid")
	}
	return nil
}

// ChangePaidRequest - payload ubah status pembayaran PO.
type ChangePaidRequest struct {
	Paid string `json:"paid"`
}

func (r ChangePaidRequest) Validate() error {
	if r.Paid != "yes" && r.Paid != "no" {
		return errors.New("status pembayaran tidak valid")
	}
	return nil
}

// UpdateInvoiceRequest - payload set nomor invoice PO.
type UpdateInvoiceRequest struct {
	Invoice string `json:"invoice"`
}

func (r UpdateInvoiceRequest) Validate() error {
	if r.Invoice == "" {
		return errors.New("nomor invoice wajib diisi")
	}
	return nil
}

// UpdateNotesRequest - payload update catatan bebas PO.
type UpdateNotesRequest struct {
	Notes string `json:"notes"`
}

// itoa kecil untuk pesan error tanpa perlu import "strconv" di top-level
// (dipisah biar file ini tetap ringan dependensinya).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}