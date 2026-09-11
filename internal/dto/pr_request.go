package dto

import (
	"errors"
	"strings"

	"rms-backend/internal/utils"
)

type CreatePRRequest struct {
	RefResponsible    utils.FlexInt              `json:"ref_responsible"`
	DescriptionItem   string                     `json:"description_item"`
	SubClient         string                     `json:"sub_client"`
	RequestedAmount   utils.FlexFloat            `json:"requested_amount"`
	PoAmount          utils.FlexFloat            `json:"po_amount"`
	Hpp               utils.FlexFloat            `json:"hpp"`
	QoutNo            string                     `json:"qout_no"`
	TargetInvoiceDate string                     `json:"target_invoice_date"`
	RefPreviousPR     *utils.FlexInt             `json:"ref_previous_pr"`   // BARU
	Payments          []CreatePRPaymentRequest   `json:"payments,omitempty"`
}

func (r *CreatePRRequest) Normalize() {
	r.DescriptionItem = strings.TrimSpace(r.DescriptionItem)
	r.SubClient = strings.TrimSpace(r.SubClient)
	r.QoutNo = strings.TrimSpace(r.QoutNo)
	for i := range r.Payments {
		r.Payments[i].Normalize()
	}
}

func (r CreatePRRequest) Validate() error {
	if int(r.RefResponsible) == 0 {
		return errors.New("responsible (CoA) wajib dipilih")
	}
	if r.DescriptionItem == "" {
		return errors.New("deskripsi item wajib diisi")
	}
	if float64(r.RequestedAmount) < 0 {
		return errors.New("jumlah permintaan tidak boleh negatif")
	}
	if float64(r.PoAmount) < 0 {
		return errors.New("po_amount tidak boleh negatif")
	}
	if float64(r.Hpp) < 0 {
		return errors.New("hpp tidak boleh negatif")
	}
	for i, p := range r.Payments {
		if err := p.Validate(); err != nil {
			return errors.New("payment ke-" + itoa(i+1) + ": " + err.Error())
		}
	}
	return nil
}

func (r CreatePRRequest) RefPreviousPRPtr() *int {
	if r.RefPreviousPR == nil {
		return nil
	}
	v := int(*r.RefPreviousPR)
	return &v
}

type UpdatePRRequest struct {
	RefResponsible    utils.FlexInt   `json:"ref_responsible"`
	DescriptionItem   string          `json:"description_item"`
	SubClient         string          `json:"sub_client"`
	RequestedAmount   utils.FlexFloat `json:"requested_amount"`
	QoutNo            string          `json:"qout_no"`
	TargetInvoiceDate string          `json:"target_invoice_date"`
	RefPreviousPR     *utils.FlexInt  `json:"ref_previous_pr"` // BARU — bisa diedit
}

func (r UpdatePRRequest) RefPreviousPRPtr() *int {
	if r.RefPreviousPR == nil {
		return nil
	}
	v := int(*r.RefPreviousPR)
	return &v
}

func (r *UpdatePRRequest) Normalize() {
	r.DescriptionItem = strings.TrimSpace(r.DescriptionItem)
	r.SubClient = strings.TrimSpace(r.SubClient)
	r.QoutNo = strings.TrimSpace(r.QoutNo)
}

func (r UpdatePRRequest) Validate() error {
	if int(r.RefResponsible) == 0 {
		return errors.New("responsible (CoA) wajib dipilih")
	}
	if r.DescriptionItem == "" {
		return errors.New("deskripsi item wajib diisi")
	}
	if float64(r.RequestedAmount) < 0 {
		return errors.New("jumlah permintaan tidak boleh negatif")
	}
	return nil
}


type ApproverAssignmentRequest struct {
	Level   utils.FlexInt `json:"level"`
	Type    string        `json:"type"`
	AdminID string        `json:"admin_id"`
}

func (r ApproverAssignmentRequest) Validate() error {
	lvl := int(r.Level)
	if lvl < 1 || lvl > 3 {
		return errors.New("level approval harus antara 1 dan 3")
	}
	if strings.TrimSpace(r.Type) == "" {
		return errors.New("tipe approval wajib diisi (mis. manager/finance/director)")
	}
	if strings.TrimSpace(r.AdminID) == "" {
		return errors.New("admin_id approver wajib diisi")
	}
	return nil
}


type SubmitPRRequest struct {
	Approvers []ApproverAssignmentRequest `json:"approvers"`
}


func (r SubmitPRRequest) Validate() error {
	if len(r.Approvers) == 0 {
		return errors.New("minimal satu approver wajib ditentukan")
	}
	if len(r.Approvers) > 3 {
		return errors.New("maksimal 3 level approver")
	}
	seenLevel := map[int]bool{}
	for i, a := range r.Approvers {
		if err := a.Validate(); err != nil {
			return errors.New("approver ke-" + itoa(i+1) + ": " + err.Error())
		}
		lvl := int(a.Level)
		if seenLevel[lvl] {
			return errors.New("level approval tidak boleh duplikat dalam satu kali submit")
		}
		seenLevel[lvl] = true
	}
	
	
	if !seenLevel[1] {
		return errors.New("approver level 1 wajib disertakan saat submit")
	}
	return nil
}

type DecidePRApprovalRequest struct {
	Notes string `json:"notes"`
}


type SetPriorityRequest struct {
	Priority string `json:"priority"`
}

var validPRPriority = map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}

func (r SetPriorityRequest) Validate() error {
	if r.Priority == "" {
		return nil
	}
	if !validPRPriority[r.Priority] {
		return errors.New("priority harus salah satu dari: low, medium, high, urgent, atau dikosongkan untuk unset")
	}
	return nil
}

type PRCommentRequest struct {
	Text string `json:"text"`
	Type string `json:"type"` // general | revision_request | internal_note, dst
}

func (r *PRCommentRequest) Normalize() {
	r.Text = strings.TrimSpace(r.Text)
	r.Type = strings.TrimSpace(r.Type)
}

func (r PRCommentRequest) Validate() error {
	if r.Text == "" {
		return errors.New("isi komentar wajib diisi")
	}
	return nil
}

type UpdatePRAmountsRequest struct {
	PoAmount utils.FlexFloat `json:"po_amount"`
	Hpp      utils.FlexFloat `json:"hpp"`
}

func (r UpdatePRAmountsRequest) Validate() error {
	if float64(r.PoAmount) < 0 {
		return errors.New("po_amount tidak boleh negatif")
	}
	if float64(r.Hpp) < 0 {
		return errors.New("hpp tidak boleh negatif")
	}
	return nil
}

type CreatePRPaymentRequest struct {
	Stage           string          `json:"stage"`
	Amount          utils.FlexFloat `json:"amount"`
	Type            string          `json:"type"`
	Bank            string          `json:"bank"`
	BankAccountNo   string          `json:"bank_account_no"`
	BankAccountName string          `json:"bank_account_name"`
	PriorityDate    string          `json:"priority_date"`
}

func (r *CreatePRPaymentRequest) Normalize() {
	r.Stage = strings.TrimSpace(r.Stage)
	r.Type = strings.TrimSpace(r.Type)
	r.Bank = strings.TrimSpace(r.Bank)
	r.BankAccountNo = strings.TrimSpace(r.BankAccountNo)
	r.BankAccountName = strings.TrimSpace(r.BankAccountName)
}

func (r CreatePRPaymentRequest) Validate() error {
	if r.Stage == "" {
		return errors.New("payment_stage wajib diisi (mis. down_payment/full_payment/final_payment)")
	}
	if r.Type == "" {
		return errors.New("payment_type wajib diisi (mis. transfer/cash/giro)")
	}
	if float64(r.Amount) <= 0 {
		return errors.New("jumlah pembayaran harus lebih dari 0")
	}
	return nil
}

type CancelPRPaymentRequest struct {
	Notes string `json:"notes"`
}

func (r CancelPRPaymentRequest) Validate() error {
	if strings.TrimSpace(r.Notes) == "" {
		return errors.New("alasan pembatalan pembayaran wajib diisi")
	}
	return nil
}

type CancelPRRequest struct {
	Notes string `json:"notes"`
}

func (r *CancelPRRequest) Normalize() {
	r.Notes = strings.TrimSpace(r.Notes)
}

func (r CancelPRRequest) Validate() error {
	if r.Notes == "" {
		return errors.New("alasan pembatalan purchase request wajib diisi")
	}
	return nil
}