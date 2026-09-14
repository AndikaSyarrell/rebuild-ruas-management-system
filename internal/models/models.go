package models

import "time"

// ---------------------------------------------------------------------
// Master / lookup tables
// ---------------------------------------------------------------------

type Region struct {
	ID         int       `json:"region_id" db:"region_id"`
	Title      string    `json:"region_title" db:"region_title"`
	CreateDate time.Time `json:"region_create_date" db:"region_create_date"`
	ModifyDate time.Time `json:"region_modify_date" db:"region_modify_date"`
}

type Division struct {
	ID        int       `json:"division_id" db:"division_id"`
	Title     string    `json:"division_title" db:"division_title"`
	Code      string    `json:"division_code" db:"division_code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Unit struct {
	ID         int       `json:"unit_id" db:"unit_id"`
	Title      string    `json:"unit_title" db:"unit_title"`
	CreateDate time.Time `json:"unit_create_date" db:"unit_create_date"`
	ModifyDate time.Time `json:"unit_modify_date" db:"unit_modify_date"`
}

type Ppn struct {
	ID         int       `json:"ppn_id" db:"ppn_id"`
	Value      float64   `json:"ppn_value" db:"ppn_value"`
	CreateDate time.Time `json:"ppn_create_date" db:"ppn_create_date"`
	ModifyDate time.Time `json:"ppn_modify_date" db:"ppn_modify_date"`
}

type Role struct {
	ID          int       `json:"role_id" db:"role_id"`
	Title       string    `json:"role_title" db:"role_title"`
	Slug        string    `json:"role_slug" db:"role_slug"`
	CreateDate  time.Time `json:"role_create_date" db:"role_create_date"`
	ModifyDate  time.Time `json:"role_modify_date" db:"role_modify_date"`
	TotalAccess int       `json:"total_access,omitempty" db:"total_access"`
}

type Access struct {
	ID         int       `json:"access_id" db:"access_id"`
	Title      string    `json:"access_title" db:"access_title"`
	Slug       string    `json:"access_slug" db:"access_slug"`
	Module     string    `json:"access_module" db:"access_module"`
	Sort       int       `json:"access_sort" db:"access_sort"`
	CreateDate time.Time `json:"access_create_date" db:"access_create_date"`
	ModifyDate time.Time `json:"access_modify_date" db:"access_modify_date"`
}

type RoleAccess struct {
	ID        int `json:"role_access_id" db:"role_access_id"`
	RefRole   int `json:"role_access_ref_role" db:"role_access_ref_role"`
	RefAccess int `json:"role_access_ref_access" db:"role_access_ref_access"`
}

// ---------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------

type Admin struct {
	ID         string    `json:"admin_id" db:"admin_id"`
	RefRegion  int       `json:"admin_ref_region" db:"admin_ref_region"`
	RefRole    *int      `json:"admin_ref_role" db:"admin_ref_role"`
	RefDivision *int     `json:"admin_ref_division" db:"admin_ref_division"`
	Token      *string   `json:"-" db:"admin_token"`
	ResetCode  *string   `json:"-" db:"admin_reset_code"`
	Email      string    `json:"admin_email" db:"admin_email"`
	Name       string    `json:"admin_name" db:"admin_name"`
	Password   *string   `json:"-" db:"admin_password"`
	Active     string    `json:"admin_active" db:"admin_active"`
	Pic        string    `json:"admin_pic" db:"admin_pic"`
	PicClient  string    `json:"admin_pic_client" db:"admin_pic_client"`
	Img        *string   `json:"admin_img" db:"admin_img"`
	ImgThumb   *string   `json:"admin_img_thmb" db:"admin_img_thmb"`
	CreateDate time.Time `json:"admin_create_date" db:"admin_create_date"`
	ModifyDate time.Time `json:"admin_modify_date" db:"admin_modify_date"`

	// Kolom hasil JOIN (opsional, diisi oleh query tertentu)
	RoleTitle   string `json:"role_title,omitempty" db:"role_title"`
	RoleSlug    string `json:"role_slug,omitempty" db:"role_slug"`
	RegionTitle string `json:"region_title,omitempty" db:"region_title"`
	DivisionTitle string `json:"division_title,omitempty" db:"division_title"`
	DivisionCode  string `json:"division_code,omitempty" db:"division_code"`
}

// AdminPublic adalah representasi Admin yang aman ditampilkan ke client (tanpa hash password/token).
type AdminPublic struct {
	ID          string `json:"admin_id"`
	Email       string `json:"admin_email"`
	Name        string `json:"admin_name"`
	Active      string `json:"admin_active"`
	RefRole     *int   `json:"admin_ref_role"`
	RoleTitle   string `json:"role_title,omitempty"`
	RefRegion   int    `json:"admin_ref_region"`
	RegionTitle string `json:"region_title,omitempty"`
	Img         string `json:"admin_img,omitempty"`
	ImgThumb    string `json:"admin_img_thmb,omitempty"`
}

func (a *Admin) ToPublic() AdminPublic {
	p := AdminPublic{
		ID:          a.ID,
		Email:       a.Email,
		Name:        a.Name,
		Active:      a.Active,
		RefRole:     a.RefRole,
		RoleTitle:   a.RoleTitle,
		RefRegion:   a.RefRegion,
		RegionTitle: a.RegionTitle,
	}
	if a.Img != nil {
		p.Img = *a.Img
	}
	if a.ImgThumb != nil {
		p.ImgThumb = *a.ImgThumb
	}
	return p
}

type Client struct {
	ID          int       `json:"client_id" db:"client_id"`
	RefRegion   int       `json:"client_ref_region" db:"client_ref_region"`
	Name        string    `json:"client_name" db:"client_name"`
	Phone       string    `json:"client_phone" db:"client_phone"`
	Email       string    `json:"client_email" db:"client_email"`
	Password    *string   `json:"-" db:"client_password"`
	ResetCode   *string   `json:"-" db:"client_reset_code"`
	Token       *string   `json:"-" db:"client_token"`
	Active      string    `json:"client_active" db:"client_active"`
	Address     string    `json:"client_address" db:"client_address"`
	CreateDate  time.Time `json:"client_create_date" db:"client_create_date"`
	ModifyDate  time.Time `json:"client_modify_date" db:"client_modify_date"`
	RegionTitle string    `json:"region_title,omitempty" db:"region_title"`
}

// ---------------------------------------------------------------------
// Transactional (PO)
// ---------------------------------------------------------------------

type PO struct {
	ID            string     `json:"po_id" db:"po_id"`
	OrderNum      string     `json:"po_order_num" db:"po_order_num"`
	Invoice       *string    `json:"po_invoice" db:"po_invoice"`
	RefClient     int        `json:"po_ref_client" db:"po_ref_client"`
	RefAdmin      string     `json:"po_ref_admin" db:"po_ref_admin"`
	RefPic        string     `json:"po_ref_pic" db:"po_ref_pic"`
	RefPicClient  *string    `json:"po_ref_pic_client" db:"po_ref_pic_client"`
	RefDivision   *int       `json:"po_ref_division" db:"po_ref_division"`
	RefRegion     int        `json:"po_ref_region" db:"po_ref_region"`
	RefPpn        int        `json:"po_ref_ppn" db:"po_ref_ppn"`
	ClientName    string     `json:"po_client_name" db:"po_client_name"`
	ClientEmail   string     `json:"po_client_email" db:"po_client_email"`
	ClientPhone   string     `json:"po_client_phone" db:"po_client_phone"`
	ClientAddr    string     `json:"po_client_address" db:"po_client_address"`
	SubClient     *string    `json:"po_subclient" db:"po_subclient"`
	Subtotal      float64    `json:"po_subtotal" db:"po_subtotal"`
	PpnRate       float64    `json:"po_ppn_rate" db:"po_ppn_rate"`
	PpnAmount     float64    `json:"po_ppn_amount" db:"po_ppn_amount"`
	Total         float64    `json:"po_total" db:"po_total"`
	ItemTotal     int        `json:"po_item_total" db:"po_item_total"`
	Status        string     `json:"po_status" db:"po_status"`
	Document      string     `json:"po_document" db:"po_document"`
	Paid          string     `json:"po_paid" db:"po_paid"`
	Notes         *string    `json:"po_notes" db:"po_notes"`
	Date          *time.Time `json:"po_date" db:"po_date"`
	ExpDate       *time.Time `json:"po_exp_date" db:"po_exp_date"`
	PreparedDate  *time.Time `json:"po_prepared_date" db:"po_prepared_date"`
	ProgressDate  *time.Time `json:"po_progress_date" db:"po_progress_date"`
	CompleteDate  *time.Time `json:"po_complete_date" db:"po_complete_date"`
	CancelDate    *time.Time `json:"po_cancel_date" db:"po_cancel_date"`
	CreateDate    time.Time  `json:"po_create_date" db:"po_create_date"`
	ModifyDate    time.Time  `json:"po_modify_date" db:"po_modify_date"`

	// hasil JOIN opsional
	AdminName     string  `json:"admin_name,omitempty" db:"admin_name"`
	PicName       string  `json:"pic_name,omitempty" db:"pic_name"`
	PicClientName string  `json:"pic_client_name,omitempty" db:"pic_client_name"`
	RegionTitle   string  `json:"region_title,omitempty" db:"region_title"`
	DivisionName  string  `json:"division_title,omitempty" db:"division_title"`
	PpnValue      float64 `json:"ppn_value,omitempty" db:"ppn_value"`
	ProductNames  string  `json:"-" db:"product_names"`
}

type POItem struct {
	ID         int       `json:"item_id" db:"item_id"`
	RefPO      string    `json:"item_ref_po" db:"item_ref_po"`
	RefUnit    int       `json:"item_ref_unit" db:"item_ref_unit"`
	Product    string    `json:"item_product" db:"item_product"`
	Desc       *string   `json:"item_desc" db:"item_desc"`
	Qty        int       `json:"item_qty" db:"item_qty"`
	Price      float64   `json:"item_price" db:"item_price"`
	CreateDate time.Time `json:"item_create_date" db:"item_create_date"`
	ModifyDate time.Time `json:"item_modify_date" db:"item_modify_date"`

	UnitTitle string `json:"unit_title,omitempty" db:"unit_title"`
}

type PODocument struct {
	ID         int       `json:"document_id" db:"document_id"`
	RefPO      string    `json:"document_ref_po" db:"document_ref_po"`
	Title      string    `json:"document_title" db:"document_title"`
	File       string    `json:"document_file" db:"document_file"`
	CreateDate time.Time `json:"document_create_date" db:"document_create_date"`
	ModifyDate time.Time `json:"document_modify_date" db:"document_modify_date"`
}

type Activity struct {
	ID         int       `json:"activity_id" db:"activity_id"`
	RefPO      string    `json:"activity_ref_po" db:"activity_ref_po"`
	RefUser    string    `json:"activity_ref_user" db:"activity_ref_user"`
	Type       string    `json:"activity_type" db:"activity_type"`
	Notes      *string   `json:"activity_notes" db:"activity_notes"`
	CreateDate time.Time `json:"activity_create_date" db:"activity_create_date"`
	ModifyDate time.Time `json:"activity_modify_date" db:"activity_modify_date"`

	AdminName string `json:"admin_name,omitempty" db:"admin_name"`
	AdminImg  string `json:"admin_img,omitempty" db:"admin_img"`
}

// Paginated adalah wrapper generik untuk hasil list + informasi paging,
// menggantikan pola PHP yang menumpangkan total_page/total_data_all pada baris pertama hasil.
type Paginated[T any] struct {
	Data      []T `json:"data"`
	TotalData int `json:"total_data"`
	TotalPage int `json:"total_page"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
	NumData   int `json:"num_data"`
}

// ---------------------------------------------------------------------
// Purchase Request (PR) module
// ---------------------------------------------------------------------

type PurchaseRequest struct {
	ID                int        `json:"pr_id" db:"pr_id"`
	RefAdmin          string     `json:"pr_ref_admin" db:"pr_ref_admin"`
	RefResponsible    int        `json:"pr_ref_responsible" db:"pr_ref_responsible"`
	PoNo          *string `json:"pr_po_no,omitempty" db:"pr_po_no"`
	RfpNo             string     `json:"pr_rfp_no" db:"pr_rfp_no"`
	DescriptionItem   string     `json:"pr_description_item" db:"pr_description_item"`
	SubClient         *string    `json:"pr_subclient" db:"pr_subclient"`
	RequestedAmount   float64    `json:"pr_requested_amount" db:"pr_requested_amount"`
	QoutNo            *string    `json:"pr_qout_no" db:"pr_qout_no"`
	PoAmount          float64    `json:"pr_po_amount" db:"pr_po_amount"`
	Hpp               float64    `json:"pr_hpp" db:"pr_hpp"`
	TargetInvoiceDate *time.Time `json:"pr_target_invoice_date" db:"pr_target_invoice_date"`
	Status            string     `json:"pr_status" db:"pr_status"`
	Priority          *string    `json:"pr_priority" db:"pr_priority"`
	PriorityRefAdmin  *string    `json:"pr_priority_ref_admin" db:"pr_priority_ref_admin"`   // BARU
	PriorityDate      *time.Time `json:"pr_priority_date" db:"pr_priority_date"`  
	CreateDate        time.Time  `json:"pr_create_date" db:"pr_create_date"`
	ModifyDate        time.Time  `json:"pr_modify_date" db:"pr_modify_date"`

	SignatureRef  *int   `json:"pr_signature_ref" db:"pr_signature_ref"`
	SignatureFile string `json:"signature_file,omitempty" db:"signature_file"` // hasil JOIN, opsional

	RefPreviousPR *int   `json:"pr_ref_previous_pr" db:"pr_ref_previous_pr"`       // BARU
	PreviousRfpNo string `json:"previous_rfp_no,omitempty" db:"previous_rfp_no"`  // BARU, hasil JOIN

	Margin           float64 `json:"pr_margin,omitempty" db:"-"`
	MarginPercentage float64 `json:"pr_margin_percentage,omitempty" db:"-"`

	AdminName       string `json:"admin_name,omitempty" db:"admin_name"`
	ResponsibleName string `json:"responsible_name,omitempty" db:"responsible_name"`
}

type AdminSignature struct {
	ID         int       `json:"signature_id" db:"signature_id"`
	RefAdmin   string    `json:"signature_ref_admin" db:"signature_ref_admin"`
	NamePic    string    `json:"signature_name_pic" db:"signature_name_pic"`
	File       string    `json:"signature_file" db:"signature_file"`
	CreateDate time.Time `json:"signature_create_date" db:"signature_create_date"`
	ModifyDate time.Time `json:"signature_modify_date" db:"signature_modify_date"`
}

type PRApproval struct {
	ID         int       `json:"approval_id" db:"approval_id"`
	RefAdmin   string    `json:"approval_ref_admin" db:"approval_ref_admin"`
	RefPR      int       `json:"approval_ref_pr" db:"approval_ref_pr"`
	Level      int       `json:"approval_level" db:"approval_level"`
	Type       string    `json:"approval_type" db:"approval_type"`
	Status     string    `json:"approval_status" db:"approval_status"`
	Round      int       `json:"approval_round" db:"approval_round"`
	Notes      *string   `json:"approval_notes" db:"approval_notes"`
	CreateDate time.Time `json:"approval_create_date" db:"approval_create_date"`

	// BARU: jejak tanda tangan yang dipakai saat approve (NULL untuk
	// reject/revision_requested/masih pending).
	SignatureRef  *int   `json:"approval_signature_ref" db:"approval_signature_ref"`
	SignatureFile string `json:"signature_file,omitempty" db:"signature_file"`

	AdminName string `json:"admin_name,omitempty" db:"admin_name"`
}

type PRStatusHistory struct {
	ID         int       `json:"history_id" db:"history_id"`
	RefAdmin   string    `json:"history_ref_admin" db:"history_ref_admin"`
	RefPR      int       `json:"history_ref_pr" db:"history_ref_pr"`
	FromStatus *string   `json:"history_from_status" db:"history_from_status"`
	ToStatus   string    `json:"history_to_status" db:"history_to_status"`
	Notes      *string   `json:"history_notes" db:"history_notes"`
	CreateDate time.Time `json:"history_create_date" db:"history_create_date"`

	AdminName string `json:"admin_name,omitempty" db:"admin_name"`
}

type PRDocument struct {
	ID         int       `json:"document_id" db:"document_id"`
	Type       string    `json:"document_type" db:"document_type"`
	FileName   string    `json:"document_file_name" db:"document_file_name"`
	FilePath   string    `json:"document_file_path" db:"document_file_path"`
	RefAdmin   string    `json:"document_ref_admin" db:"document_ref_admin"`
	RefPR      int       `json:"document_ref_pr" db:"document_ref_pr"`
	CreateDate time.Time `json:"document_create_date" db:"document_create_date"`

	AdminName string `json:"admin_name,omitempty" db:"admin_name"`
}

type PRComment struct {
	ID         int       `json:"comment_id" db:"comment_id"`
	RefAdmin   string    `json:"comment_ref_admin" db:"comment_ref_admin"`
	RefPR      int       `json:"comment_ref_pr" db:"comment_ref_pr"`
	Text       string    `json:"comment_text" db:"comment_text"`
	Type       string    `json:"comment_type" db:"comment_type"`
	CreateDate time.Time `json:"comment_create_date" db:"comment_create_date"`

	AdminName string `json:"admin_name,omitempty" db:"admin_name"`
}

type PRPayment struct {
	ID              int        `json:"payment_id" db:"payment_id"`
	RefPR           int        `json:"payment_ref_pr" db:"payment_ref_pr"`
	RefAdminInput   string     `json:"payment_ref_admin_input" db:"payment_ref_admin_input"`
	Stage           string     `json:"payment_stage" db:"payment_stage"`
	Amount          float64    `json:"payment_amount" db:"payment_amount"`
	Type            string     `json:"payment_type" db:"payment_type"`
	Bank            *string    `json:"payment_bank" db:"payment_bank"`
	BankAccountNo   *string    `json:"payment_bank_account_no" db:"payment_bank_account_no"`
	BankAccountName *string    `json:"payment_bank_account_name" db:"payment_bank_account_name"`
	PriorityDate    *time.Time `json:"payment_priority_date" db:"payment_priority_date"`
	Status          string     `json:"payment_status" db:"payment_status"`
	PaidDate        *time.Time `json:"payment_paid_date" db:"payment_paid_date"`
	RefAdminPaid    *string    `json:"payment_ref_admin_paid" db:"payment_ref_admin_paid"`
	CreateDate      time.Time  `json:"payment_create_date" db:"payment_create_date"`
	ModifyDate      time.Time  `json:"payment_modify_date" db:"payment_modify_date"`

	AdminInputName string `json:"admin_input_name,omitempty" db:"admin_input_name"`
	AdminPaidName  string `json:"admin_paid_name,omitempty" db:"admin_paid_name"`
}

type Responsible struct {
	ID         int       `json:"responsible_id" db:"responsible_id"`
	Name       string    `json:"responsible_name" db:"responsible_name"`
	CoaCode    string    `json:"responsible_coa_code" db:"responsible_coa_code"`
	Status     string    `json:"responsible_status" db:"responsible_status"` // active | inactive
	CreateDate time.Time `json:"responsible_create_date" db:"responsible_create_date"`
	ModifyDate time.Time `json:"responsible_modify_date" db:"responsible_modify_date"`
}