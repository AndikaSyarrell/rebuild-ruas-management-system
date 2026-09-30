package dto

import (
	"errors"
	"strings"
	"fmt"

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
	RefPreviousPR     *utils.FlexInt             `json:"ref_previous_pr"`   
	Payments          []PRPaymentDraftRequest    `json:"payments,omitempty"`
	Priority 		  string					 `json:"priority"`	
	ConfirmJoinQuotation bool 					 `json:"confirm_join_quotation"`
	PoNo 			  string 					 `json:"po_no"`
}

func (r *CreatePRRequest) Normalize() {
	r.DescriptionItem = strings.TrimSpace(r.DescriptionItem)
	r.SubClient = strings.TrimSpace(r.SubClient)
	r.QoutNo = strings.TrimSpace(r.QoutNo)
	r.Priority = strings.ToLower(strings.TrimSpace(r.Priority))
	r.PoNo = strings.TrimSpace(r.PoNo)
	if r.Priority == ""{
		r.Priority = defaultPRPriority
	}
	for i := range r.Payments {
		r.Payments[i].Normalize()
	}
}

func (r CreatePRRequest) Validate() error {
	if int(r.RefResponsible) == 0 {
		return errors.New("responsible (CoA) wajib dipilih")
	}
	if len(r.PoNo) > 196 {
		return errors.New("nomor po maksimal 196 karakter")
	}
	if !validPRPriority[r.Priority]{
		return errors.New("Priority harus salah satu dari low, medium, atau high")
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
	RefResponsible       utils.FlexInt            `json:"ref_responsible"`
	DescriptionItem      string                   `json:"description_item"`
	SubClient            string                   `json:"sub_client"`
	RequestedAmount      utils.FlexFloat          `json:"requested_amount"`
	QoutNo               string                   `json:"qout_no"`
	TargetInvoiceDate    string                   `json:"target_invoice_date"`
	RefPreviousPR        *utils.FlexInt           `json:"ref_previous_pr"`
	Payments             []PRPaymentDraftRequest  `json:"payments,omitempty"`
	ConfirmJoinQuotation bool                     `json:"confirm_join_quotation"` // BARU
	PoNo 				*string `json:"po_no"`
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
	if r.PoNo != nil {
		v := strings.TrimSpace(*r.PoNo)
		r.PoNo = &v
	}
	for i := range r.Payments { // BARU
		r.Payments[i].Normalize()
	}
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
	if r.PoNo != nil && len(*r.PoNo) > 196 {
		return errors.New("nomor po maksimal 196 karakter")
	}
	for i, p := range r.Payments{
		if err := p.Validate(); err != nil {
			return errors.New("payment ke-" + itoa(i+1) + ":" + err.Error())
		}
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
	CheckerID   string        `json:"checker_id"`
	SignatureID utils.FlexInt `json:"signature_id"`
}


func (r SubmitPRRequest) Validate() error {
	if strings.TrimSpace(r.CheckerID) == "" {
		return errors.New("checker_id wajib diisi")
	}
	if int(r.SignatureID) == 0 {
		return errors.New("signature_id wajib diisi")
	}
	return nil
}

type DecidePRApprovalRequest struct {
	Notes string `json:"notes"`
}

const defaultPRPriority = "medium"

var validPRPriority = map[string]bool{"low": true, "medium": true, "high": true}

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
	PoNo     string          `json:"po_no"`
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

type PRPaymentDraftRequest struct {
	Stage           string          `json:"stage"`
	Amount          utils.FlexFloat `json:"amount"`
	Type            string          `json:"type"`
	Bank            string          `json:"bank"`
	BankAccountNo   string          `json:"bank_account_no"`
	BankAccountName string          `json:"bank_account_name"`
}

func (r *PRPaymentDraftRequest) Normalize() {
	r.Stage = strings.TrimSpace(r.Stage)
	r.Type = strings.TrimSpace(r.Type)
	r.Bank = strings.TrimSpace(r.Bank)
	r.BankAccountNo = strings.TrimSpace(r.BankAccountNo)
	r.BankAccountName = strings.TrimSpace(r.BankAccountName)
}

func (r PRPaymentDraftRequest) Validate() error {
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

type CreatePRPaymentRequest struct {
	PRPaymentDraftRequest
	PriorityDate string `json:"priority_date"`
}

func (r *CreatePRPaymentRequest) Normalize() {
	r.PRPaymentDraftRequest.Normalize()
	r.PriorityDate = strings.TrimSpace(r.PriorityDate)
}

// SetPaymentPriorityDateRequest: "" = hapus tanggal prioritas.
type SetPaymentPriorityDateRequest struct {
	PriorityDate string `json:"priority_date"`
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

func validateApproverAssignments(approvers []ApproverAssignmentRequest) error {
	if len(approvers) == 0 {
		return errors.New("minimal satu approver wajib ditentukan")
	}
	if len(approvers) > 3 {
		return errors.New("maksimal 3 level approver")
	}
	seenLevel := map[int]bool{}
	for i, a := range approvers {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("approver ke-%d: %w", i+1, err)
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

type BulkSubmitPRItem struct {
	PRID      utils.FlexInt `json:"pr_id"`
	CheckerID string        `json:"checker_id,omitempty"` // override default_checker_id
}

type BulkSubmitPRRequest struct {
	SignatureID      utils.FlexInt      `json:"signature_id"`
	DefaultCheckerID string             `json:"default_checker_id,omitempty"`
	Items            []BulkSubmitPRItem `json:"items"`
}

func (r BulkSubmitPRRequest) Validate() error {
	if int(r.SignatureID) == 0 {
		return errors.New("signature_id wajib diisi")
	}
	if len(r.Items) == 0 {
		return errors.New("items wajib diisi minimal 1 purchase request")
	}
	if len(r.Items) > 50 {
		return errors.New("maksimal 50 purchase request per bulk submit")
	}
	seenPR := map[int]bool{}
	for i, it := range r.Items {
		id := int(it.PRID)
		if id == 0 {
			return fmt.Errorf("item ke-%d: pr_id wajib diisi", i+1)
		}
		if seenPR[id] {
			return fmt.Errorf("item ke-%d: pr_id %d duplikat dalam satu batch", i+1, id)
		}
		seenPR[id] = true
		if strings.TrimSpace(it.CheckerID) == "" && strings.TrimSpace(r.DefaultCheckerID) == "" {
			return fmt.Errorf("item ke-%d (pr_id=%d): checker_id atau default_checker_id wajib diisi", i+1, id)
		}
	}
	return nil
}

// --- Bulk Decide (Approve/Reject) ---

type BulkApprovalItem struct {
	ApprovalID utils.FlexInt `json:"approval_id"`
	Notes      string        `json:"notes,omitempty"`
}

type BulkDecideApprovalRequest struct {
	Decision string              `json:"decision"` // "approved" | "rejected"
	Items    []BulkApprovalItem  `json:"items"`
}

func (r BulkDecideApprovalRequest) Validate() error {
	if r.Decision != "approved" && r.Decision != "rejected" {
		return errors.New("decision harus 'approved' atau 'rejected'")
	}
	if len(r.Items) == 0 {
		return errors.New("items wajib diisi minimal 1 approval")
	}
	if len(r.Items) > 50 {
		return errors.New("maksimal 50 approval per bulk decide")
	}
	seen := map[int]bool{}
	for i, it := range r.Items {
		id := int(it.ApprovalID)
		if id == 0 {
			return fmt.Errorf("item ke-%d: approval_id wajib diisi", i+1)
		}
		if seen[id] {
			return fmt.Errorf("item ke-%d: approval_id %d duplikat dalam satu batch", i+1, id)
		}
		seen[id] = true
	}
	return nil
}