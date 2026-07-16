package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
)

type RoleRepo struct{ db *sql.DB }

func NewRoleRepo(db *sql.DB) *RoleRepo { return &RoleRepo{db: db} }

func (r *RoleRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Role, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Role`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := `SELECT ro.role_id, ro.role_title, ro.role_slug, ro.role_create_date, ro.role_modify_date,
			  COUNT(ra.role_access_id) AS total_access
			  FROM T_Role ro LEFT JOIN T_Role_Access ra ON ro.role_id = ra.role_access_ref_role
			  GROUP BY ro.role_id
			  ORDER BY total_access DESC, ro.role_create_date ASC
			  LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Role
	for rows.Next() {
		var v models.Role
		if err := rows.Scan(&v.ID, &v.Title, &v.Slug, &v.CreateDate, &v.ModifyDate, &v.TotalAccess); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *RoleRepo) ListSelect(ctx context.Context) ([]models.Role, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT role_id, role_title, role_slug, role_create_date, role_modify_date FROM T_Role ORDER BY role_title ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Role
	for rows.Next() {
		var v models.Role
		if err := rows.Scan(&v.ID, &v.Title, &v.Slug, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RoleRepo) GetByID(ctx context.Context, id int) (*models.Role, error) {
	var v models.Role
	err := r.db.QueryRowContext(ctx,
		`SELECT role_id, role_title, role_slug, role_create_date, role_modify_date FROM T_Role WHERE role_id = ?`, id).
		Scan(&v.ID, &v.Title, &v.Slug, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *RoleRepo) Create(ctx context.Context, title, slug string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Role (role_title, role_slug, role_create_date) VALUES (?, ?, NOW())`, title, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RoleRepo) Update(ctx context.Context, id int, title, slug string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Role SET role_title = ?, role_slug = ? WHERE role_id = ?`, title, slug, id)
	return err
}

// Delete menghapus role. Baris T_Role_Access & admin_ref_role terkait
// otomatis tertangani oleh FK (CASCADE untuk role_access, SET NULL untuk admin).
func (r *RoleRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Role WHERE role_id = ?`, id)
	return err
}
