package dto

import (
	"time"
	"strings"

	"rms-backend/internal/models"
	"rms-backend/internal/service"
	"rms-backend/internal/repository"
)

type PRResponse struct {
	ID                int     `json:"pr_id"`
	RefAdmin          string  `json:"pr_ref_admin"`
	AdminName         string  `json:"admin_name,omitempty"`
	RefResponsible    int     `json:"pr_ref_responsible"`
	ResponsibleName   string  `json:"responsible_name,omitempty"`
	RfpNo             string  `json:"pr_rfp_no"`
	RfpDate           string  `json:"pr_rfp_date"`
	DescriptionItem   string  `json:"pr_description_item"`
	SubClient         string  `json:"pr_subclient,omitempty"`
	RequestedAmount   float64 `json:"pr_requested_amount"`
	QoutNo            string  `json:"pr_qout_no,omitempty"`
	PoAmount          float64 `json:"pr_po_amount"`
	Hpp               float64 `json:"pr_hpp"`
	TargetInvoiceDate string  `json:"pr_target_invoice_date,omitempty"`
	Status            string  `json:"pr_status"`
	Priority          string  `json:"pr_priority,omitempty"`
	PriorityRefAdmin  string  `json:"pr_priority_ref_admin,omitempty"`
	PriorityDate      string  `json:"pr_priority_date,omitempty"`
	CreateDate        string  `json:"pr_create_date"`
	ModifyDate        string  `json:"pr_modify_date"`
	PoNoDisplay 	  string  `json:"pr_po_no_display"`

	RefPreviousPR 	  *int    `json:"pr_ref_previous_pr,omitempty"` 
	PreviousRfpNo     string  `json:"previous_rfp_no,omitempty"`    
	RefQuotation 	  *int    `json:"pr_ref_quotation,omitempty"`
	QuotationNo  	  string  `json:"quotation_no,omitempty"`
	PoNo         	  string  `json:"pr_po_no,omitempty"`          
	Margin            float64 `json:"pr_margin"`
	MarginPercentage  float64 `json:"pr_margin_percentage"`
}

func NewPRResponse(m models.PurchaseRequest) PRResponse {
	subClient := ""
	if m.SubClient != nil {
		subClient = *m.SubClient
	}
	poNoDisplay := ""
	if m.PoNo != nil {
		poNoDisplay = strings.TrimSpace(*m.PoNo)
	}
	if poNoDisplay == "" {
		poNoDisplay = service.PoNotReleasedLabel
	}
	qoutNo := ""
	if m.QoutNo != nil {
		qoutNo = *m.QoutNo
	}
	priority := ""
	if m.Priority != nil {
		priority = *m.Priority
	}
	priorityRefAdmin := ""
	if m.PriorityRefAdmin != nil {
		priorityRefAdmin = *m.PriorityRefAdmin
	}
	poNo := ""
	if m.PoNo != nil {
		poNo = *m.PoNo
	}

	margin := m.PoAmount - m.Hpp
	marginPct := 0.0
	if m.PoAmount != 0 {
		marginPct = (margin / m.PoAmount) * 100
	}

	return PRResponse{
		ID:                m.ID,
		RefAdmin:          m.RefAdmin,
		AdminName:         m.AdminName,
		RefResponsible:    m.RefResponsible,
		ResponsibleName:   m.ResponsibleName,
		RfpNo:             m.RfpNo,
		DescriptionItem:   m.DescriptionItem,
		SubClient:         subClient,
		RequestedAmount:   m.RequestedAmount,
		QoutNo:            qoutNo,
		PoAmount:          m.PoAmount,
		Hpp:               m.Hpp,
		TargetInvoiceDate: formatDate(m.TargetInvoiceDate),
		RefPreviousPR: 	   m.RefPreviousPR,
		PreviousRfpNo:	   m.PreviousRfpNo,
		RefQuotation:      m.RefQuotation,
		QuotationNo:       m.QuotationNo,
		PoNo:              poNo,
		Status:            m.Status,
		Priority:          priority,
		PriorityRefAdmin:  priorityRefAdmin,
		PriorityDate:      formatDateTime(m.PriorityDate),
		CreateDate:        m.CreateDate.Format(time.RFC3339),
		ModifyDate:        m.ModifyDate.Format(time.RFC3339),
		Margin:            margin,
		MarginPercentage:  marginPct,
	}
}

func NewPRResponseList(list []models.PurchaseRequest) []PRResponse {
	out := make([]PRResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRResponse(m))
	}
	return out
}

type QuotationConflictResponse struct {
	QuotationID int                        `json:"quotation_id"`
	QuotationNo string                     `json:"quotation_no"`
	LinkedPO    string                     `json:"linked_po,omitempty"` 	
	Members     []QuotationMemberResponse  `json:"members"`
}

type QuotationMemberResponse struct {
	PRID   int    `json:"pr_id"`
	RfpNo  string `json:"rfp_no"`
	Status string `json:"pr_status"`
}

func NewQuotationConflictResponse(c *service.QuotationConflict) QuotationConflictResponse {
	members := make([]QuotationMemberResponse, 0, len(c.Members))
	for _, m := range c.Members {
		members = append(members, QuotationMemberResponse{PRID: m.PRID, RfpNo: m.RfpNo, Status: m.Status})
	}
	return QuotationConflictResponse{
		QuotationID: c.QuotationID,
		QuotationNo: c.QuotationNo,
		LinkedPO:    c.LinkedPO,
		Members:     members,
	}
}

type PRApprovalResponse struct {
	ID         int    `json:"approval_id"`
	RefAdmin   string `json:"approval_ref_admin"`
	AdminName  string `json:"admin_name,omitempty"`
	RefPR      int    `json:"approval_ref_pr"`
	Level      int    `json:"approval_level"`
	Type       string `json:"approval_type"`
	Status     string `json:"approval_status"`
	Round      int    `json:"approval_round"`
	Notes      string `json:"approval_notes,omitempty"`
	DecidedDate string `json:"approval_decided_date,omitempty"`
	CreateDate string `json:"approval_create_date"`
}

func NewPRApprovalResponse(m models.PRApproval) PRApprovalResponse {
	notes := ""
	if m.Notes != nil {
		notes = *m.Notes
	}
	refAdmin := ""
	if m.RefAdmin != nil {
		refAdmin = *m.RefAdmin
	}
	return PRApprovalResponse{
		ID: m.ID, RefAdmin: refAdmin, AdminName: m.AdminName, RefPR: m.RefPR,
		Level: m.Level, Type: m.Type, Status: m.Status, Round: m.Round,
		Notes: notes, CreateDate: m.CreateDate.Format(time.RFC3339),
		DecidedDate: formatDateTime(m.DecidedDate),
	}
}

type PRDecisionHistoryResponse struct {
	ApprovalID      int     `json:"approval_id"`
	PRID            int     `json:"pr_id"`
	RfpNo           string  `json:"rfp_no"`
	DescriptionItem string  `json:"pr_description_item"`
	RequestedAmount float64 `json:"pr_requested_amount"`
	RequesterName   string  `json:"requester_name,omitempty"`
	PRStatus        string  `json:"pr_status"`
	Level           int     `json:"approval_level"`
	Type            string  `json:"approval_type"`
	Decision        string  `json:"decision"`
	Notes           string  `json:"approval_notes,omitempty"`
	Round           int     `json:"approval_round"`
	DecidedDate     string  `json:"decided_date,omitempty"`
}

func NewPRDecisionHistoryResponseList(list []repository.DecidedApproval) []PRDecisionHistoryResponse {
	out := make([]PRDecisionHistoryResponse, 0, len(list))
	for _, d := range list {
		notes := ""
		if d.Notes != nil {
			notes = *d.Notes
		}
		out = append(out, PRDecisionHistoryResponse{
			ApprovalID: d.ID, PRID: d.RefPR, RfpNo: d.RfpNo, DescriptionItem: d.DescriptionItem,
			RequestedAmount: d.RequestedAmount, RequesterName: d.RequesterName, PRStatus: d.PRStatus,
			Level: d.Level, Type: d.Type, Decision: d.Status, Notes: notes, Round: d.Round,
			DecidedDate: formatDateTime(d.DecidedDate),
		})
	}
	return out
}

func NewPRApprovalResponseList(list []models.PRApproval) []PRApprovalResponse {
	out := make([]PRApprovalResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRApprovalResponse(m))
	}
	return out
}

type PRHistoryResponse struct {
	ID         int    `json:"history_id"`
	RefAdmin   string `json:"history_ref_admin"`
	AdminName  string `json:"admin_name,omitempty"`
	RefPR      int    `json:"history_ref_pr"`
	FromStatus string `json:"history_from_status,omitempty"`
	ToStatus   string `json:"history_to_status"`
	Notes      string `json:"history_notes,omitempty"`
	CreateDate string `json:"history_create_date"`
}

func NewPRHistoryResponse(m models.PRStatusHistory) PRHistoryResponse {
	from := ""
	if m.FromStatus != nil {
		from = *m.FromStatus
	}
	notes := ""
	if m.Notes != nil {
		notes = *m.Notes
	}
	return PRHistoryResponse{
		ID: m.ID, RefAdmin: m.RefAdmin, AdminName: m.AdminName, RefPR: m.RefPR,
		FromStatus: from, ToStatus: m.ToStatus, Notes: notes,
		CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewPRHistoryResponseList(list []models.PRStatusHistory) []PRHistoryResponse {
	out := make([]PRHistoryResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRHistoryResponse(m))
	}
	return out
}

type PRCommentResponse struct {
	ID         int    `json:"comment_id"`
	RefAdmin   string `json:"comment_ref_admin"`
	AdminName  string `json:"admin_name,omitempty"`
	RefPR      int    `json:"comment_ref_pr"`
	Text       string `json:"comment_text"`
	Type       string `json:"comment_type"`
	CreateDate string `json:"comment_create_date"`
}

func NewPRCommentResponse(m models.PRComment) PRCommentResponse {
	return PRCommentResponse{
		ID: m.ID, RefAdmin: m.RefAdmin, AdminName: m.AdminName, RefPR: m.RefPR,
		Text: m.Text, Type: m.Type, CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewPRCommentResponseList(list []models.PRComment) []PRCommentResponse {
	out := make([]PRCommentResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRCommentResponse(m))
	}
	return out
}

type PRPaymentResponse struct {
	ID              int     `json:"payment_id"`
	RefPR           int     `json:"payment_ref_pr"`
	RefAdminInput   string  `json:"payment_ref_admin_input"`
	AdminInputName  string  `json:"admin_input_name,omitempty"`
	Stage           string  `json:"payment_stage"`
	Amount          float64 `json:"payment_amount"`
	Type            string  `json:"payment_type"`
	Bank            string  `json:"payment_bank,omitempty"`
	BankAccountNo   string  `json:"payment_bank_account_no,omitempty"`
	BankAccountName string  `json:"payment_bank_account_name,omitempty"`
	PriorityDate    string  `json:"payment_priority_date,omitempty"`
	Status          string  `json:"payment_status"`
	PaidDate        string  `json:"payment_paid_date,omitempty"`
	RefAdminPaid    string  `json:"payment_ref_admin_paid,omitempty"`
	AdminPaidName   string  `json:"admin_paid_name,omitempty"`
	CreateDate      string  `json:"payment_create_date"`
}

func NewPRPaymentResponse(m models.PRPayment) PRPaymentResponse {
	bank, bankNo, bankName, refAdminPaid := "", "", "", ""
	if m.Bank != nil {
		bank = *m.Bank
	}
	if m.BankAccountNo != nil {
		bankNo = *m.BankAccountNo
	}
	if m.BankAccountName != nil {
		bankName = *m.BankAccountName
	}
	if m.RefAdminPaid != nil {
		refAdminPaid = *m.RefAdminPaid
	}
	return PRPaymentResponse{
		ID: m.ID, RefPR: m.RefPR, RefAdminInput: m.RefAdminInput, AdminInputName: m.AdminInputName,
		Stage: m.Stage, Amount: m.Amount, Type: m.Type,
		Bank: bank, BankAccountNo: bankNo, BankAccountName: bankName,
		PriorityDate: formatDate(m.PriorityDate), Status: m.Status,
		PaidDate: formatDateTime(m.PaidDate), RefAdminPaid: refAdminPaid, AdminPaidName: m.AdminPaidName,
		CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewPRPaymentResponseList(list []models.PRPayment) []PRPaymentResponse {
	out := make([]PRPaymentResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRPaymentResponse(m))
	}
	return out
}