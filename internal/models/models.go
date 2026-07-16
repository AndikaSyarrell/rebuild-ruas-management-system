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
	ID           string     `json:"po_id" db:"po_id"`
	OrderNum     string     `json:"po_order_num" db:"po_order_num"`
	Invoice      *string    `json:"po_invoice" db:"po_invoice"`
	RefClient    int        `json:"po_ref_client" db:"po_ref_client"`
	RefAdmin     string     `json:"po_ref_admin" db:"po_ref_admin"`
	RefPic       string     `json:"po_ref_pic" db:"po_ref_pic"`
	RefDivision  *int       `json:"po_ref_division" db:"po_ref_division"`
	RefRegion    int        `json:"po_ref_region" db:"po_ref_region"`
	RefPpn       int        `json:"po_ref_ppn" db:"po_ref_ppn"`
	ClientName   string     `json:"po_client_name" db:"po_client_name"`
	ClientEmail  string     `json:"po_client_email" db:"po_client_email"`
	ClientPhone  string     `json:"po_client_phone" db:"po_client_phone"`
	ClientAddr   string     `json:"po_client_address" db:"po_client_address"`
	SubClient    *string    `json:"po_subclient" db:"po_subclient"`
	Subtotal     float64    `json:"po_subtotal" db:"po_subtotal"`
	PpnRate      float64    `json:"po_ppn_rate" db:"po_ppn_rate"`
	PpnAmount    float64    `json:"po_ppn_amount" db:"po_ppn_amount"`
	Total        float64    `json:"po_total" db:"po_total"`
	ItemTotal    int        `json:"po_item_total" db:"po_item_total"`
	Status       string     `json:"po_status" db:"po_status"`
	Document     string     `json:"po_document" db:"po_document"`
	Paid         string     `json:"po_paid" db:"po_paid"`
	Notes        *string    `json:"po_notes" db:"po_notes"`
	Date         *time.Time `json:"po_date" db:"po_date"`
	ExpDate      *time.Time `json:"po_exp_date" db:"po_exp_date"`
	PreparedDate *time.Time `json:"po_prepared_date" db:"po_prepared_date"`
	ProgressDate *time.Time `json:"po_progress_date" db:"po_progress_date"`
	CompleteDate *time.Time `json:"po_complete_date" db:"po_complete_date"`
	CancelDate   *time.Time `json:"po_cancel_date" db:"po_cancel_date"`
	CreateDate   time.Time  `json:"po_create_date" db:"po_create_date"`
	ModifyDate   time.Time  `json:"po_modify_date" db:"po_modify_date"`

	// hasil JOIN opsional
	AdminName    string  `json:"admin_name,omitempty" db:"admin_name"`
	PicName      string  `json:"pic_name,omitempty" db:"pic_name"`
	RegionTitle  string  `json:"region_title,omitempty" db:"region_title"`
	DivisionName string  `json:"division_title,omitempty" db:"division_title"`
	PpnValue     float64 `json:"ppn_value,omitempty" db:"ppn_value"`
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
