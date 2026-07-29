package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"rms-backend/internal/models"
)

type PORepo struct{ db *sql.DB }

func NewPORepo(db *sql.DB) *PORepo { return &PORepo{db: db} }

func (r *PORepo) OrderNumExists(ctx context.Context, orderNum string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM T_Po WHERE po_order_num = ? LIMIT 1`, orderNum).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *PORepo) IDExists(ctx context.Context, id string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM T_Po WHERE po_id = ? LIMIT 1`, id).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// NewPOInput adalah payload pembuatan PO beserta baris item.
type NewPOItemInput struct {
	Desc    string
	Product string
	Qty     int
	UnitID  int
	Price   float64
}

type NewPOInput struct {
	ID          string
	OrderNum    string
	RegionID    int
	AdminID     string // pembuat (po_ref_admin)
	PicID       string // PIC penanganan (po_ref_pic)
	DivisionID  *int
	PpnID       int
	PpnRate     float64
	Date        string // Y-m-d
	ClientID    int
	ClientName  string
	ClientEmail string
	ClientPhone string
	ClientAddr  string
	SubClient   string
	Items       []NewPOItemInput
}

// Create menyimpan PO + item dalam SATU transaksi database (memperbaiki bug PHP di
// controller_create.php yang insert item satu-per-satu tanpa transaksi, sehingga PO
// bisa "setengah jadi" bila salah satu insert item gagal di tengah jalan).
func (r *PORepo) Create(ctx context.Context, in NewPOInput) error {
	subtotal := 0.0
	for _, it := range in.Items {
		subtotal += it.Price * float64(it.Qty)
	}
	ppnAmount := subtotal * in.PpnRate / 100
	total := subtotal + ppnAmount

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO T_Po (po_id, po_order_num, po_ref_client, po_ref_admin, po_ref_pic, po_ref_region,
		 po_ref_division, po_ref_ppn, po_client_name, po_client_email, po_client_phone, po_client_address,
		 po_subclient, po_subtotal, po_ppn_rate, po_ppn_amount, po_total, po_item_total, po_status, po_date,
		 po_create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?, NOW())`,
		in.ID, in.OrderNum, in.ClientID, in.AdminID, in.PicID, in.RegionID, in.DivisionID, in.PpnID,
		in.ClientName, in.ClientEmail, in.ClientPhone, in.ClientAddr, in.SubClient,
		subtotal, in.PpnRate, ppnAmount, total, len(in.Items), in.Date)
	if err != nil {
		return err
	}

	if len(in.Items) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO T_Po_Item (item_ref_po, item_ref_unit, item_product, item_desc, item_qty, item_price, item_create_date)
			VALUES (?, ?, ?, ?, ?, ?, NOW())`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, it := range in.Items {
			if _, err := stmt.ExecContext(ctx, in.ID, it.UnitID, it.Product, it.Desc, it.Qty, it.Price); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

const poDetailSelect = `
	SELECT po.po_id, po.po_order_num, po.po_invoice, po.po_ref_client, po.po_ref_admin, po.po_ref_pic,
	       po.po_ref_division, po.po_ref_region, po.po_ref_ppn, po.po_client_name, po.po_client_email,
	       po.po_client_phone, po.po_client_address, po.po_subclient, po.po_subtotal, po.po_ppn_rate,
	       po.po_ppn_amount, po.po_total, po.po_item_total, po.po_status, po.po_document, po.po_paid,
	       po.po_notes, po.po_date, po.po_exp_date, po.po_prepared_date, po.po_progress_date,
	       po.po_complete_date, po.po_cancel_date, po.po_create_date, po.po_modify_date,
	       COALESCE(ad.admin_name, ''), COALESCE(pic.admin_name, ''), COALESCE(rg.region_title, ''),
	       COALESCE(dv.division_title, ''), COALESCE(pp.ppn_value, 0)
	FROM T_Po po
	LEFT JOIN T_Admin ad ON po.po_ref_admin = ad.admin_id
	LEFT JOIN T_Admin pic ON po.po_ref_pic = pic.admin_id
	LEFT JOIN T_Region rg ON po.po_ref_region = rg.region_id
	LEFT JOIN T_Division dv ON po.po_ref_division = dv.division_id
	LEFT JOIN T_Ppn pp ON po.po_ref_ppn = pp.ppn_id
`

func scanPO(row interface {
	Scan(dest ...interface{}) error
}) (*models.PO, error) {
	var v models.PO
	err := row.Scan(&v.ID, &v.OrderNum, &v.Invoice, &v.RefClient, &v.RefAdmin, &v.RefPic,
		&v.RefDivision, &v.RefRegion, &v.RefPpn, &v.ClientName, &v.ClientEmail,
		&v.ClientPhone, &v.ClientAddr, &v.SubClient, &v.Subtotal, &v.PpnRate,
		&v.PpnAmount, &v.Total, &v.ItemTotal, &v.Status, &v.Document, &v.Paid,
		&v.Notes, &v.Date, &v.ExpDate, &v.PreparedDate, &v.ProgressDate,
		&v.CompleteDate, &v.CancelDate, &v.CreateDate, &v.ModifyDate,
		&v.AdminName, &v.PicName, &v.RegionTitle, &v.DivisionName, &v.PpnValue)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PORepo) GetDetail(ctx context.Context, id string) (*models.PO, error) {
	row := r.db.QueryRowContext(ctx, poDetailSelect+" WHERE po.po_id = ?", id)
	return scanPO(row)
}

// ListFilter membawa seluruh kriteria pencarian daftar PO (menggantikan 9+ parameter
// posisional pada PO::get_data di PHP).
type ListFilter struct {
	Page      int
	PerPage   int
	Status    string
	Keyword   string
	StartDate string
	EndDate   string
	RegionID  int
	SortBy    string // "old" | "new"
	PicID     string // dibatasi ke admin tertentu (acsg_pic)
}

func (r *PORepo) List(ctx context.Context, f ListFilter) ([]models.PO, int, error) {
	var conds []string
	var args []interface{}

	conds = append(conds, "po_status = ?")
	args = append(args, f.Status)

	if f.Keyword != "" {
		conds = append(conds, `(po.po_id LIKE ? OR po.po_order_num LIKE ? OR cl.client_name LIKE ? OR po.po_subclient LIKE ?
			OR EXISTS (SELECT 1 FROM T_Po_Item it WHERE it.item_ref_po = po.po_id AND it.item_product LIKE ?))`)
		like := "%" + f.Keyword + "%"
		args = append(args, like, like, like, like, like)
	}
	if f.StartDate != "" && f.EndDate != "" {
		conds = append(conds, "po.po_date BETWEEN ? AND ?")
		args = append(args, f.StartDate, f.EndDate)
	}
	if f.RegionID != 0 {
		conds = append(conds, "po.po_ref_region = ?")
		args = append(args, f.RegionID)
	}
	if f.PicID != "" {
		conds = append(conds, "po.po_ref_pic = ?")
		args = append(args, f.PicID)
	}

	whereClause := "WHERE " + strings.Join(conds, " AND ")

	countQuery := `SELECT COUNT(DISTINCT po.po_id) FROM T_Po po LEFT JOIN T_Client cl ON po.po_ref_client = cl.client_id ` + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortClause := "ORDER BY po.po_date ASC"
	if f.SortBy == "new" {
		sortClause = "ORDER BY po.po_date DESC"
	}

	perPage := f.PerPage
	if perPage <= 0 {
		perPage = 10
	}
	offset := (f.Page - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	listQuery := `
		SELECT po.po_id, po.po_order_num, po.po_invoice, po.po_status, po.po_total, po.po_document,
		       po.po_date, po.po_exp_date, po.po_paid, po.po_subclient,
		       COALESCE(cl.client_name, ''), COALESCE(ad.admin_name, ''), COALESCE(dv.division_title, ''),
		       COALESCE(rg.region_title, '')
		FROM T_Po po
		LEFT JOIN T_Client cl ON po.po_ref_client = cl.client_id
		LEFT JOIN T_Admin ad ON po.po_ref_pic = ad.admin_id
		LEFT JOIN T_Region rg ON po.po_ref_region = rg.region_id
		LEFT JOIN T_Division dv ON po.po_ref_division = dv.division_id
		` + whereClause + `
		` + sortClause + `
		LIMIT ? OFFSET ?`

	args = append(args, perPage, offset)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.PO
	for rows.Next() {
		var v models.PO
		if err := rows.Scan(&v.ID, &v.OrderNum, &v.Invoice, &v.Status, &v.Total, &v.Document,
			&v.Date, &v.ExpDate, &v.Paid, &v.SubClient,
			&v.ClientName, &v.PicName, &v.DivisionName, &v.RegionTitle); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

// CountByStatus mengganti PO::total_data_all_status - dipakai untuk badge dashboard.
func (r *PORepo) CountByStatus(ctx context.Context, regionID int, picID string) (map[string]int, error) {
	var conds []string
	var args []interface{}
	if regionID != 0 {
		conds = append(conds, "po_ref_region = ?")
		args = append(args, regionID)
	}
	if picID != "" {
		conds = append(conds, "po_ref_pic = ?")
		args = append(args, picID)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	query := fmt.Sprintf(`SELECT po_status, COUNT(*) FROM T_Po %s GROUP BY po_status`, where)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{"open": 0, "progress": 0, "prepared": 0, "complete": 0, "cancel": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}

func (r *PORepo) CountByPaid(ctx context.Context, regionID int, picID string) (map[string]int, error) {
	var conds []string
	var args []interface{}
	if regionID != 0 {
		conds = append(conds, "po_ref_region = ?")
		args = append(args, regionID)
	}
	if picID != "" {
		conds = append(conds, "po_ref_pic = ?")
		args = append(args, picID)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	query := fmt.Sprintf(`SELECT po_paid, COUNT(*) FROM T_Po %s GROUP BY po_paid`, where)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{"yes": 0, "no": 0}
	for rows.Next() {
		var paid string
		var count int
		if err := rows.Scan(&paid, &count); err != nil {
			return nil, err
		}
		out[paid] = count
	}
	return out, rows.Err()
}

func (r *PORepo) ChangeStatus(ctx context.Context, id, status string) error {
	dateColumn := map[string]string{
		"progress": "po_progress_date",
		"prepared": "po_prepared_date",
		"complete": "po_complete_date",
		"cancel":   "po_cancel_date",
	}
	col, hasDate := dateColumn[status]

	query := "UPDATE T_Po SET po_status = ?"
	if hasDate {
		query += fmt.Sprintf(", %s = NOW()", col)
	}
	query += " WHERE po_id = ?"

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PORepo) ChangePaid(ctx context.Context, id, paid string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Po SET po_paid = ? WHERE po_id = ?`, paid, id)
	return err
}

func (r *PORepo) UpdateInvoice(ctx context.Context, id, invoice string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Po SET po_invoice = ? WHERE po_id = ?`, invoice, id)
	return err
}

func (r *PORepo) UpdateNotes(ctx context.Context, id, notes string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Po SET po_notes = ? WHERE po_id = ?`, notes, id)
	return err
}

func (r *PORepo) UpdateDocInfo(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Po SET po_document = ? WHERE po_id = ?`, status, id)
	return err
}

type UpdatePOInput struct {
	OrderNum    string
	RegionID    int
	PicID       string
	DivisionID  *int
	Date        string
	ClientID    int
	ClientName  string
	ClientEmail string
	ClientPhone string
	ClientAddr  string
	SubClient   string
}

func (r *PORepo) UpdateHeader(ctx context.Context, id string, in UpdatePOInput) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE T_Po SET po_order_num = ?, po_ref_region = ?, po_ref_pic = ?, po_ref_division = ?, po_date = ?,
		 po_ref_client = ?, po_client_name = ?, po_client_email = ?, po_client_phone = ?, po_client_address = ?,
		 po_subclient = ?
		WHERE po_id = ?`,
		in.OrderNum, in.RegionID, in.PicID, in.DivisionID, in.Date,
		in.ClientID, in.ClientName, in.ClientEmail, in.ClientPhone, in.ClientAddr, in.SubClient, id)
	return err
}

// RecomputeTotals menghitung ulang subtotal/ppn/total berdasarkan item yang ada
// SECARA ATOMIK di sisi database (menggantikan pola PHP: fetch semua item ke PHP,
// jumlahkan di PHP, lalu UPDATE - rawan race condition bila 2 request bersamaan).
func (r *PORepo) RecomputeTotals(ctx context.Context, poID string, ppnRate float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var subtotal float64
	var itemCount int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(item_price * item_qty), 0), COUNT(*) FROM T_Po_Item WHERE item_ref_po = ?`, poID).
		Scan(&subtotal, &itemCount)
	if err != nil {
		return err
	}

	ppnAmount := subtotal * ppnRate / 100
	total := subtotal + ppnAmount

	_, err = tx.ExecContext(ctx,
		`UPDATE T_Po SET po_subtotal = ?, po_ppn_amount = ?, po_total = ?, po_item_total = ? WHERE po_id = ?`,
		subtotal, ppnAmount, total, itemCount, poID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PORepo) Delete(ctx context.Context, id string) error {
	// FK ON DELETE CASCADE pada T_Po_Item, T_Document, T_Activity menangani cleanup otomatis.
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Po WHERE po_id = ?`, id)
	return err
}

// --- Items ---

func (r *PORepo) ListItems(ctx context.Context, poID string) ([]models.POItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT it.item_id, it.item_ref_po, it.item_ref_unit, it.item_product, it.item_desc, it.item_qty,
		       it.item_price, it.item_create_date, it.item_modify_date, COALESCE(u.unit_title, '')
		FROM T_Po_Item it
		LEFT JOIN T_Unit u ON it.item_ref_unit = u.unit_id
		WHERE it.item_ref_po = ?
		ORDER BY it.item_create_date ASC`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.POItem
	for rows.Next() {
		var v models.POItem
		if err := rows.Scan(&v.ID, &v.RefPO, &v.RefUnit, &v.Product, &v.Desc, &v.Qty, &v.Price,
			&v.CreateDate, &v.ModifyDate, &v.UnitTitle); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PORepo) AddItem(ctx context.Context, poID string, unitID int, product, desc string, qty int, price float64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Po_Item (item_ref_po, item_ref_unit, item_product, item_desc, item_qty, item_price, item_create_date)
		VALUES (?, ?, ?, ?, ?, ?, NOW())`, poID, unitID, product, desc, qty, price)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PORepo) UpdateItem(ctx context.Context, itemID int, unitID int, product, desc string, qty int, price float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE T_Po_Item SET item_ref_unit = ?, item_product = ?, item_desc = ?, item_qty = ?, item_price = ?
		WHERE item_id = ?`, unitID, product, desc, qty, price, itemID)
	return err
}

// GetItemPO mengembalikan po_id pemilik sebuah item, dipakai untuk memvalidasi
// bahwa item yang diedit/dihapus benar-benar milik PO yang diklaim caller.
func (r *PORepo) GetItemPO(ctx context.Context, itemID int) (string, error) {
	var poID string
	err := r.db.QueryRowContext(ctx, `SELECT item_ref_po FROM T_Po_Item WHERE item_id = ?`, itemID).Scan(&poID)
	return poID, err
}

func (r *PORepo) DeleteItem(ctx context.Context, itemID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Po_Item WHERE item_id = ?`, itemID)
	return err
}

// --- Reporting ---

func (r *PORepo) ReportChart(ctx context.Context, startDate, endDate string, regionID int, picID, status string) ([]map[string]interface{}, error) {
	var conds []string
	conds = append(conds, "po_id != ''")
	var args []interface{}

	if status != "" && status != "all" {
		conds = append(conds, "po_status = ?")
		args = append(args, status)
	} else {
		conds = append(conds, "po_status != 'cancel'")
	}
	if regionID != 0 {
		conds = append(conds, "po_ref_region = ?")
		args = append(args, regionID)
	}
	if picID != "" {
		conds = append(conds, "po_ref_pic = ?")
		args = append(args, picID)
	}
	if startDate != "" && endDate != "" {
		conds = append(conds, "po_date BETWEEN ? AND ?")
		args = append(args, startDate, endDate)
	}

	query := `SELECT DATE_FORMAT(po_date, '%Y-%m-%d') AS d, COUNT(po_id) AS total
			  FROM T_Po WHERE ` + strings.Join(conds, " AND ") + `
			  GROUP BY d ORDER BY d ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var d string
		var total int
		if err := rows.Scan(&d, &total); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"date": d, "total": total})
	}
	return out, rows.Err()
}

func (r *PORepo) ReportBarsByRegion(ctx context.Context, startDate, endDate, status string) ([]map[string]interface{}, error) {
	conds := []string{"po_id != ''"}
	var args []interface{}
	if status != "" && status != "all" {
		conds = append(conds, "po_status = ?")
		args = append(args, status)
	} else {
		conds = append(conds, "po_status != 'cancel'")
	}
	if startDate != "" && endDate != "" {
		conds = append(conds, "po_date BETWEEN ? AND ?")
		args = append(args, startDate, endDate)
	}

	query := `SELECT rg.region_id, rg.region_title, COALESCE(SUM(po.po_total), 0) AS total_amount
			  FROM T_Po po LEFT JOIN T_Region rg ON rg.region_id = po.po_ref_region
			  WHERE ` + strings.Join(conds, " AND ") + `
			  GROUP BY po.po_ref_region ORDER BY rg.region_title ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var regionID int
		var title string
		var total float64
		if err := rows.Scan(&regionID, &title, &total); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"region_id": regionID, "region_title": title, "total_amount": total})
	}
	return out, rows.Err()
}

func (r *PORepo) StatByRegion(ctx context.Context, picID string) ([]map[string]interface{}, error) {
	where := ""
	var args []interface{}
	if picID != "" {
		where = "WHERE pr.po_ref_pic = ?"
		args = append(args, picID)
	}

	query := `SELECT rg.region_id, rg.region_title, COUNT(pr.po_id) AS total_po,
			  SUM(CASE WHEN pr.po_status='open' THEN 1 ELSE 0 END),
			  SUM(CASE WHEN pr.po_status='prepared' THEN 1 ELSE 0 END),
			  SUM(CASE WHEN pr.po_status='progress' THEN 1 ELSE 0 END),
			  SUM(CASE WHEN pr.po_status='complete' THEN 1 ELSE 0 END),
			  SUM(CASE WHEN pr.po_status='cancel' THEN 1 ELSE 0 END)
			  FROM T_Po pr LEFT JOIN T_Region rg ON pr.po_ref_region = rg.region_id
			  ` + where + `
			  GROUP BY pr.po_ref_region ORDER BY rg.region_title ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var regionID sql.NullInt64
		var title string
		var total, open, prepared, progress, complete, cancel int
		if err := rows.Scan(&regionID, &title, &total, &open, &prepared, &progress, &complete, &cancel); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"region_id": regionID.Int64, "region_title": title, "total_po": total,
			"total_open": open, "total_prepared": prepared, "total_progress": progress,
			"total_complete": complete, "total_cancel": cancel,
		})
	}
	return out, rows.Err()
}

// --- Export (breakdown per item) ---

// ExportFilter membawa seluruh kriteria filter export: rentang tanggal, divisi,
// status PO, region, dan client. Kosong/0 berarti "semua" untuk dimensi tsb.
type ExportFilter struct {
	StartDate  string
	EndDate    string
	DivisionID int
	Status     string
	RegionID   int
	ClientID   int
}

// ExportRow adalah 1 baris hasil export = 1 item PO (join header PO + item).
// Semua field bertipe primitif (string/float64/int) karena NULL sudah
// di-COALESCE di level SQL - lebih sederhana dipakai langsung oleh excelize.
type ExportRow struct {
	OrderNum     string
	Invoice      string
	Status       string
	Date         string // format YYYY-MM-DD, "" jika belum diisi
	RegionTitle  string
	DivisionName string
	PicName      string
	AdminName    string
	ClientName   string
	ClientEmail  string
	ClientPhone  string
	SubClient    string
	ItemProduct  string
	ItemDesc     string
	UnitTitle    string
	ItemQty      int
	ItemPrice    float64
	Subtotal     float64
	PpnRate      float64
	PpnAmount    float64
	Total        float64
	Paid         string
	Notes        string
}

// ExportItemRows mengganti PO::get_data_export dari PHP: satu baris per item PO,
// dengan seluruh join & kondisi filter ter-parameterisasi penuh.
func (r *PORepo) ExportItemRows(ctx context.Context, f ExportFilter) ([]ExportRow, error) {
	conds := []string{"po.po_id != ''"}
	var args []interface{}

	if f.StartDate != "" && f.EndDate != "" {
		conds = append(conds, "po.po_date BETWEEN ? AND ?")
		args = append(args, f.StartDate, f.EndDate)
	}
	if f.DivisionID != 0 {
		conds = append(conds, "po.po_ref_division = ?")
		args = append(args, f.DivisionID)
	}
	if f.Status != "" && f.Status != "all" {
		conds = append(conds, "po.po_status = ?")
		args = append(args, f.Status)
	}
	if f.RegionID != 0 {
		conds = append(conds, "po.po_ref_region = ?")
		args = append(args, f.RegionID)
	}
	if f.ClientID != 0 {
		conds = append(conds, "po.po_ref_client = ?")
		args = append(args, f.ClientID)
	}

	query := `
		SELECT po.po_order_num, COALESCE(po.po_invoice, ''), po.po_status,
		       COALESCE(DATE_FORMAT(po.po_date, '%Y-%m-%d'), ''),
		       COALESCE(rg.region_title, ''), COALESCE(dv.division_title, ''),
		       COALESCE(pic.admin_name, ''), COALESCE(ad.admin_name, ''),
		       po.po_client_name, po.po_client_email, po.po_client_phone,
		       COALESCE(po.po_subclient, ''),
		       it.item_product, COALESCE(it.item_desc, ''), COALESCE(u.unit_title, ''),
		       it.item_qty, it.item_price,
		       po.po_subtotal, po.po_ppn_rate, po.po_ppn_amount, po.po_total,
		       po.po_paid, COALESCE(po.po_notes, '')
		FROM T_Po po
		JOIN T_Po_Item it ON it.item_ref_po = po.po_id
		LEFT JOIN T_Region rg ON po.po_ref_region = rg.region_id
		LEFT JOIN T_Division dv ON po.po_ref_division = dv.division_id
		LEFT JOIN T_Admin pic ON po.po_ref_pic = pic.admin_id
		LEFT JOIN T_Admin ad ON po.po_ref_admin = ad.admin_id
		LEFT JOIN T_Unit u ON it.item_ref_unit = u.unit_id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY po.po_date ASC, po.po_id ASC, it.item_id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportRow
	for rows.Next() {
		var v ExportRow
		if err := rows.Scan(&v.OrderNum, &v.Invoice, &v.Status, &v.Date,
			&v.RegionTitle, &v.DivisionName, &v.PicName, &v.AdminName,
			&v.ClientName, &v.ClientEmail, &v.ClientPhone, &v.SubClient,
			&v.ItemProduct, &v.ItemDesc, &v.UnitTitle, &v.ItemQty, &v.ItemPrice,
			&v.Subtotal, &v.PpnRate, &v.PpnAmount, &v.Total, &v.Paid, &v.Notes); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PORepo) ListDashboardOpen(ctx context.Context, regionID int, picID string, limit int) ([]models.PO, error) {
	conds := []string{"po.po_status = 'open'"}
	var args []interface{}
	if regionID != 0 {
		conds = append(conds, "po.po_ref_region = ?")
		args = append(args, regionID)
	}
	if picID != "" {
		conds = append(conds, "po.po_ref_pic = ?")
		args = append(args, picID)
	}

	query := `SELECT po.po_id, po.po_order_num, po.po_status, po.po_total, po.po_date, po.po_exp_date,
			  po.po_subclient, COALESCE(cl.client_name, ''), COALESCE(ad.admin_name, ''), COALESCE(rg.region_title, '')
			  FROM T_Po po
			  LEFT JOIN T_Client cl ON po.po_ref_client = cl.client_id
			  LEFT JOIN T_Admin ad ON po.po_ref_pic = ad.admin_id
			  LEFT JOIN T_Region rg ON po.po_ref_region = rg.region_id
			  WHERE ` + strings.Join(conds, " AND ") + `
			  LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PO
	for rows.Next() {
		var v models.PO
		if err := rows.Scan(&v.ID, &v.OrderNum, &v.Status, &v.Total, &v.Date, &v.ExpDate,
			&v.SubClient, &v.ClientName, &v.PicName, &v.RegionTitle); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
