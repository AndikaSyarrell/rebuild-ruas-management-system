package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
	"rms-backend/internal/db"
)

type AccessRepo struct{ db db.Querier }

func NewAccessRepo(db db.Querier) *AccessRepo { return &AccessRepo{db: db} }

// HasAccess mengimplementasikan middleware.AccessChecker. Menggantikan query PHP
// yang JOIN ke AT_Access (nama tabel lama) dengan T_Role_Access yang sudah ternormalisasi
// dan sepenuhnya parameterized.
func (r *AccessRepo) HasAccess(ctx context.Context, adminID string, slug string) (bool, error) {
	query := `SELECT 1
			  FROM T_Admin a
			  JOIN T_Role_Access ra ON ra.role_access_ref_role = a.admin_ref_role
			  JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
			  WHERE a.admin_id = ? AND ac.access_slug = ? AND a.admin_active = 'active'
			  LIMIT 1`
	var dummy int
	err := r.db.QueryRowContext(ctx, query, adminID, slug).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *AccessRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Access, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Access`).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		`SELECT access_id, access_title, access_slug, access_module, access_sort, access_create_date, access_modify_date
		 FROM T_Access ORDER BY access_module ASC, access_create_date ASC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Access
	for rows.Next() {
		var v models.Access
		if err := rows.Scan(&v.ID, &v.Title, &v.Slug, &v.Module, &v.Sort, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *AccessRepo) ListByModule(ctx context.Context, module string) ([]models.Access, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT access_id, access_title, access_slug, access_module, access_sort, access_create_date, access_modify_date
		 FROM T_Access WHERE access_module = ? ORDER BY access_sort ASC, access_title ASC`, module)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Access
	for rows.Next() {
		var v models.Access
		if err := rows.Scan(&v.ID, &v.Title, &v.Slug, &v.Module, &v.Sort, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *AccessRepo) GetByID(ctx context.Context, id int) (*models.Access, error) {
	var v models.Access
	err := r.db.QueryRowContext(ctx,
		`SELECT access_id, access_title, access_slug, access_module, access_sort, access_create_date, access_modify_date
		 FROM T_Access WHERE access_id = ?`, id).
		Scan(&v.ID, &v.Title, &v.Slug, &v.Module, &v.Sort, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AccessRepo) Create(ctx context.Context, title, module, slug string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Access (access_title, access_module, access_slug, access_create_date) VALUES (?, ?, ?, NOW())`,
		title, module, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *AccessRepo) Update(ctx context.Context, id int, title, module, slug string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Access SET access_title = ?, access_module = ?, access_slug = ? WHERE access_id = ?`,
		title, module, slug, id)
	return err
}

func (r *AccessRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Access WHERE access_id = ?`, id)
	return err
}

// --- Role <-> Access pivot ---

func (r *AccessRepo) GetAccessIDsForRole(ctx context.Context, roleID int) ([]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT role_access_ref_access FROM T_Role_Access WHERE role_access_ref_role = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReplaceRoleAccess mengganti seluruh daftar akses milik sebuah role dalam satu
// transaksi (menggantikan pola PHP: delete_at_by_role lalu loop insert_at satu-per-satu).
func (r *AccessRepo) ReplaceRoleAccess(ctx context.Context, roleID int, accessIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM T_Role_Access WHERE role_access_ref_role = ?`, roleID); err != nil {
		return err
	}

	if len(accessIDs) > 0 {
		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO T_Role_Access (role_access_ref_role, role_access_ref_access, role_access_create_date) VALUES (?, ?, NOW())`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, aid := range accessIDs {
			if _, err := stmt.ExecContext(ctx, roleID, aid); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *AccessRepo) ListSlugsForRole(ctx context.Context, roleID int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ac.access_slug
		FROM T_Role_Access ra
		JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
		WHERE ra.role_access_ref_role = ?
		ORDER BY ac.access_slug ASC`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}
