package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
	"rms-backend/internal/utils"
	"rms-backend/internal/db"
)

type AdminRepo struct{ db db.Querier }

func NewAdminRepo(db db.Querier) *AdminRepo { return &AdminRepo{db: db} }

const adminSelectWithJoins = `
	SELECT a.admin_id, a.admin_ref_region, a.admin_ref_role, a.admin_token, a.admin_reset_code,
	       a.admin_email, a.admin_name, a.admin_password, a.admin_active, a.admin_pic, a.admin_pic_client,
	       a.admin_img, a.admin_img_thmb, a.admin_create_date, a.admin_modify_date,
	       COALESCE(ro.role_title, ''), COALESCE(ro.role_slug, ''), COALESCE(rg.region_title, '')
	FROM T_Admin a
	LEFT JOIN T_Role ro ON a.admin_ref_role = ro.role_id
	LEFT JOIN T_Region rg ON a.admin_ref_region = rg.region_id
`

func scanAdmin(row interface {
	Scan(dest ...interface{}) error
}) (*models.Admin, error) {
	var v models.Admin
	err := row.Scan(&v.ID, &v.RefRegion, &v.RefRole, &v.Token, &v.ResetCode,
		&v.Email, &v.Name, &v.Password, &v.Active, &v.Pic, &v.PicClient,
		&v.Img, &v.ImgThumb, &v.CreateDate, &v.ModifyDate,
		&v.RoleTitle, &v.RoleSlug, &v.RegionTitle)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AdminRepo) GetByEmail(ctx context.Context, email string) (*models.Admin, error) {
	row := r.db.QueryRowContext(ctx, adminSelectWithJoins+" WHERE a.admin_email = ? LIMIT 1", email)
	return scanAdmin(row)
}

func (r *AdminRepo) GetByID(ctx context.Context, id string) (*models.Admin, error) {
	row := r.db.QueryRowContext(ctx, adminSelectWithJoins+" WHERE a.admin_id = ? LIMIT 1", id)
	return scanAdmin(row)
}

func (r *AdminRepo) GetByToken(ctx context.Context, token string) (*models.Admin, error) {
	row := r.db.QueryRowContext(ctx, adminSelectWithJoins+" WHERE a.admin_token = ? LIMIT 1", utils.HashToken(token))
	return scanAdmin(row)
}

func (r *AdminRepo) GetByResetCode(ctx context.Context, code string) (*models.Admin, error) {
	row := r.db.QueryRowContext(ctx, adminSelectWithJoins+" WHERE a.admin_reset_code = ? LIMIT 1", utils.HashToken(code))
	return scanAdmin(row)
}

func (r *AdminRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM T_Admin WHERE admin_email = ? LIMIT 1`, email).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *AdminRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Admin, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Admin`).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		adminSelectWithJoins+" ORDER BY a.admin_email ASC, a.admin_create_date ASC LIMIT ? OFFSET ?", perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Admin
	for rows.Next() {
		v, err := scanAdmin(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

// ListPIC mengganti Admin::get_admin_pic (kandidat PIC internal, bukan PIC client).
func (r *AdminRepo) ListPIC(ctx context.Context) ([]models.Admin, error) {
	rows, err := r.db.QueryContext(ctx,
		adminSelectWithJoins+" WHERE a.admin_pic = 'yes' AND a.admin_pic_client = 'no' ORDER BY a.admin_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Admin
	for rows.Next() {
		v, err := scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// ListPICClient mengganti Admin::get_admin_pic_client.
func (r *AdminRepo) ListPICClient(ctx context.Context) ([]models.Admin, error) {
	rows, err := r.db.QueryContext(ctx,
		adminSelectWithJoins+" WHERE a.admin_pic_client = 'yes' ORDER BY a.admin_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Admin
	for rows.Next() {
		v, err := scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// ListByAccessSlug mengganti Admin::get_list_picker / get_list_email (admin aktif
// yang role-nya memiliki akses tertentu).
func (r *AdminRepo) ListByAccessSlug(ctx context.Context, slug string) ([]models.Admin, error) {
	query := adminSelectWithJoins + `
		JOIN T_Role_Access ra ON ra.role_access_ref_role = a.admin_ref_role
		JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
		WHERE ac.access_slug = ? AND a.admin_active = 'active'
		GROUP BY a.admin_id
		ORDER BY a.admin_name ASC`
	rows, err := r.db.QueryContext(ctx, query, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Admin
	for rows.Next() {
		v, err := scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *AdminRepo) Create(ctx context.Context, id, email, name string, regionID int, roleID *int, token string, pic, picClient string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Admin (admin_id, admin_email, admin_name, admin_ref_role, admin_token, admin_active,
		 admin_pic, admin_pic_client, admin_ref_region, admin_create_date)
		 VALUES (?, ?, ?, ?, ?, 'inactive', ?, ?, ?, NOW())`,
		id, email, name, roleID, utils.HashToken(token), pic, picClient, regionID)
	return err
}

func (r *AdminRepo) Update(ctx context.Context, id string, roleID *int, email, name, pic, picClient string, regionID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Admin SET admin_email = ?, admin_name = ?, admin_ref_role = ?, admin_pic = ?, admin_pic_client = ?, admin_ref_region = ?
		 WHERE admin_id = ?`,
		email, name, roleID, pic, picClient, regionID, id)
	return err
}

func (r *AdminRepo) UpdateImages(ctx context.Context, id, img, imgThumb string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Admin SET admin_img = ?, admin_img_thmb = ? WHERE admin_id = ?`, img, imgThumb, id)
	return err
}

func (r *AdminRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Admin SET admin_password = ? WHERE admin_id = ?`, passwordHash, id)
	return err
}

func (r *AdminRepo) UpdateToken(ctx context.Context, id, token string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Admin SET admin_token = ? WHERE admin_id = ?`, utils.HashToken(token), id)
	return err
}

func (r *AdminRepo) SetResetCode(ctx context.Context, email, code string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Admin SET admin_reset_code = ? WHERE admin_email = ?`, utils.HashToken(code), email)
	return err
}

func (r *AdminRepo) ActivateAccount(ctx context.Context, email, token, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Admin SET admin_token = NULL, admin_active = 'active', admin_password = ?
		 WHERE admin_email = ? AND admin_token = ?`, passwordHash, email, utils.HashToken(token))
	return err
}

func (r *AdminRepo) SetNewPassword(ctx context.Context, email, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Admin SET admin_password = ?, admin_reset_code = NULL WHERE admin_email = ?`, passwordHash, email)
	return err
}

func (r *AdminRepo) ChangeActive(ctx context.Context, id, active string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Admin SET admin_active = ?, admin_token = NULL WHERE admin_id = ?`, active, id)
	return err
}

func (r *AdminRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Admin WHERE admin_id = ?`, id)
	return err
}
