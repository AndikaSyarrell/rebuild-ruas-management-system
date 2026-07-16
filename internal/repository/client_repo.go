package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
)

type ClientRepo struct{ db *sql.DB }

func NewClientRepo(db *sql.DB) *ClientRepo { return &ClientRepo{db: db} }

const clientSelectWithJoin = `
	SELECT c.client_id, c.client_ref_region, c.client_name, c.client_phone, c.client_email,
	       c.client_password, c.client_reset_code, c.client_token, c.client_active, c.client_address,
	       c.client_create_date, c.client_modify_date, COALESCE(rg.region_title, '')
	FROM T_Client c
	LEFT JOIN T_Region rg ON c.client_ref_region = rg.region_id
`

func scanClient(row interface {
	Scan(dest ...interface{}) error
}) (*models.Client, error) {
	var v models.Client
	err := row.Scan(&v.ID, &v.RefRegion, &v.Name, &v.Phone, &v.Email, &v.Password, &v.ResetCode, &v.Token,
		&v.Active, &v.Address, &v.CreateDate, &v.ModifyDate, &v.RegionTitle)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ClientRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM T_Client WHERE client_email = ? LIMIT 1`, email).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ClientRepo) GetByEmail(ctx context.Context, email string) (*models.Client, error) {
	row := r.db.QueryRowContext(ctx, clientSelectWithJoin+" WHERE c.client_email = ? LIMIT 1", email)
	return scanClient(row)
}

func (r *ClientRepo) GetByID(ctx context.Context, id int) (*models.Client, error) {
	row := r.db.QueryRowContext(ctx, clientSelectWithJoin+" WHERE c.client_id = ? LIMIT 1", id)
	return scanClient(row)
}

func (r *ClientRepo) Search(ctx context.Context, keyword string) ([]models.Client, error) {
	rows, err := r.db.QueryContext(ctx,
		clientSelectWithJoin+` WHERE (? = '' OR c.client_name LIKE CONCAT('%', ?, '%'))
		ORDER BY c.client_name ASC, c.client_create_date DESC`, keyword, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Client
	for rows.Next() {
		v, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *ClientRepo) ListPaged(ctx context.Context, page, perPage, regionID int, keyword string) ([]models.Client, int, error) {
	whereRegion := regionID == 0
	whereKeyword := keyword == ""

	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Client WHERE (? OR client_ref_region = ?) AND (? OR client_name LIKE CONCAT('%', ?, '%'))`,
		whereRegion, regionID, whereKeyword, keyword).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := clientSelectWithJoin + `
		WHERE (? OR c.client_ref_region = ?) AND (? OR c.client_name LIKE CONCAT('%', ?, '%'))
		ORDER BY c.client_create_date DESC, c.client_name ASC
		LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, whereRegion, regionID, whereKeyword, keyword, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Client
	for rows.Next() {
		v, err := scanClient(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

func (r *ClientRepo) Create(ctx context.Context, name, phone, email, address string, regionID int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Client (client_name, client_phone, client_email, client_address, client_ref_region,
		 client_password, client_create_date) VALUES (?, ?, ?, ?, ?, '', NOW())`,
		name, phone, email, address, regionID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ClientRepo) Update(ctx context.Context, id int, name, phone, email, address string, regionID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Client SET client_name = ?, client_phone = ?, client_email = ?, client_address = ?, client_ref_region = ?
		 WHERE client_id = ?`, name, phone, email, address, regionID, id)
	return err
}

func (r *ClientRepo) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Client SET client_password = ? WHERE client_id = ?`, passwordHash, id)
	return err
}

func (r *ClientRepo) ChangeActive(ctx context.Context, id int, active string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Client SET client_active = ? WHERE client_id = ?`, active, id)
	return err
}

func (r *ClientRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Client WHERE client_id = ?`, id)
	return err
}
