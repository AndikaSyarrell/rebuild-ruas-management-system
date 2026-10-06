package dto

import (
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
)

// --- GET /api/pr/po-options ---

type POLinkOptionResponse struct {
	ID                 string   `json:"po_id"`
	OrderNum           string   `json:"po_order_num"`
	Status             string   `json:"po_status"`
	SubClient          string   `json:"po_subclient,omitempty"`
	ClientName         string   `json:"client_name,omitempty"`
	QuotHint           string   `json:"po_quot_no,omitempty"`
	LinkedQuotationNos []string `json:"linked_quotation_nos"`
}

func NewPOLinkOptionResponseList(list []repository.POLinkOption) []POLinkOptionResponse {
	out := make([]POLinkOptionResponse, 0, len(list))
	for _, o := range list {
		out = append(out, POLinkOptionResponse{
			ID: o.ID, OrderNum: o.OrderNum, Status: o.Status, SubClient: o.SubClient,
			ClientName: o.ClientName, QuotHint: o.QuotHint,
			LinkedQuotationNos: splitProductNames(o.LinkedQuotationNos),
		})
	}
	return out
}

// --- GET /api/pr/link-check ---

type LinkCheckPOResponse struct {
	ID                 string   `json:"po_id"`
	OrderNum           string   `json:"po_order_num"`
	Status             string   `json:"po_status"`
	SubClient          string   `json:"po_subclient,omitempty"`
	LinkedQuotationNos []string `json:"linked_quotation_nos"`
}

type LinkCheckQuotationResponse struct {
	Exists      bool                      `json:"exists"`
	QuotationID int                       `json:"quotation_id,omitempty"`
	QuotationNo string                    `json:"quotation_no"`
	LinkedPO    *LinkCheckPOResponse      `json:"linked_po"`
	Members     []QuotationMemberResponse `json:"members"`
}

type ResponsibleRefResponse struct {
	ID      int    `json:"responsible_id"`
	Name    string `json:"responsible_name"`
	CoaCode string `json:"responsible_coa_code"`
}

type ResponsibleLockResponse struct {
	Locked      bool                     `json:"locked"`
	Conflict    bool                     `json:"conflict"`
	Source      string                   `json:"source"`
	Responsible *ResponsibleRefResponse  `json:"responsible"`
	Candidates  []ResponsibleRefResponse `json:"candidates"`
}

type LinkCheckResponse struct {
	Level       string                     `json:"level"`
	Code        string                     `json:"code"`
	Message     string                     `json:"message"`
	Quotation   LinkCheckQuotationResponse `json:"quotation"`
	PO          *LinkCheckPOResponse       `json:"po"`
	Responsible *ResponsibleLockResponse   `json:"responsible"`
	PoNoDisplay string                     `json:"po_no_display"`
}

func newLinkCheckPO(p *service.LinkCheckPO) *LinkCheckPOResponse {
	if p == nil {
		return nil
	}
	nos := p.LinkedQuotationNos
	if nos == nil {
		nos = []string{}
	}
	return &LinkCheckPOResponse{
		ID: p.ID, OrderNum: p.OrderNum, Status: p.Status, SubClient: p.SubClient, LinkedQuotationNos: nos,
	}
}

func NewLinkCheckResponse(r *service.LinkCheckResult) LinkCheckResponse {
	members := make([]QuotationMemberResponse, 0, len(r.Members))
	for _, m := range r.Members {
		members = append(members, QuotationMemberResponse{PRID: m.PRID, RfpNo: m.RfpNo, Status: m.Status})
	}
	out := LinkCheckResponse{
		Level: r.Level, Code: r.Code, Message: r.Message,
		Quotation: LinkCheckQuotationResponse{
			Exists: r.QuotationExists, QuotationID: r.QuotationID, QuotationNo: r.QuotationNo,
			LinkedPO: newLinkCheckPO(r.QuotationLinkedPO), Members: members,
		},
		PO: newLinkCheckPO(r.PO),
		PoNoDisplay: r.PoNoDisplay,
	}
	if r.Responsible != nil {
		items := make([]ResponsibleRefResponse, 0, len(r.Responsible.Items))
		for _, it := range r.Responsible.Items {
			items = append(items, ResponsibleRefResponse{ID: it.ID, Name: it.Name, CoaCode: it.CoaCode})
		}
		lock := &ResponsibleLockResponse{
			Locked: r.Responsible.Locked, Conflict: r.Responsible.Conflict,
			Source: r.Responsible.Source, Candidates: []ResponsibleRefResponse{},
		}
		if r.Responsible.Locked && len(items) == 1 {
			lock.Responsible = &items[0]
		}
		if r.Responsible.Conflict {
			lock.Candidates = items
		}
		out.Responsible = lock
	}
	return out
}
