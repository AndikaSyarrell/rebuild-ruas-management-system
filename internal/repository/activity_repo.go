package repository

import (
	"context"
	// "database/sql"

	"rms-backend/internal/models"
	"rms-backend/internal/db"
)

type ActivityRepo struct{ db db.Querier }

func NewActivityRepo(db db.Querier) *ActivityRepo { return &ActivityRepo{db: db} }

// ListByPO mengambil seluruh histori aktivitas sebuah PO, terurut terbaru dahulu
// (menggantikan PHP yang mengelompokkan per-tanggal dengan 1+N query - di sini cukup 1 query).
func (r *ActivityRepo) ListByPO(ctx context.Context, poID string) ([]models.Activity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ac.activity_id, ac.activity_ref_po, ac.activity_ref_user, ac.activity_type, ac.activity_notes,
		       ac.activity_create_date, ac.activity_modify_date, COALESCE(ad.admin_name, ''), COALESCE(ad.admin_img, '')
		FROM T_Activity ac
		LEFT JOIN T_Admin ad ON ac.activity_ref_user = ad.admin_id
		WHERE ac.activity_ref_po = ?
		ORDER BY ac.activity_create_date DESC`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Activity
	for rows.Next() {
		var v models.Activity
		if err := rows.Scan(&v.ID, &v.RefPO, &v.RefUser, &v.Type, &v.Notes,
			&v.CreateDate, &v.ModifyDate, &v.AdminName, &v.AdminImg); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListDashboard mengambil aktivitas status-changing terbaru lintas PO untuk feed dashboard.
func (r *ActivityRepo) ListDashboard(ctx context.Context, page, perPage int) ([]models.Activity, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Activity WHERE activity_type IN ('open','progress','prepared','complete','cancel')`).
		Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx, `
		SELECT ac.activity_id, ac.activity_ref_po, ac.activity_ref_user, ac.activity_type, ac.activity_notes,
		       ac.activity_create_date, ac.activity_modify_date, COALESCE(ad.admin_name, ''), COALESCE(ad.admin_img, '')
		FROM T_Activity ac
		LEFT JOIN T_Admin ad ON ac.activity_ref_user = ad.admin_id
		WHERE ac.activity_type IN ('open','progress','prepared','complete','cancel')
		ORDER BY ac.activity_create_date DESC
		LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Activity
	for rows.Next() {
		var v models.Activity
		if err := rows.Scan(&v.ID, &v.RefPO, &v.RefUser, &v.Type, &v.Notes,
			&v.CreateDate, &v.ModifyDate, &v.AdminName, &v.AdminImg); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *ActivityRepo) Insert(ctx context.Context, poID, userID, actType, notes string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Activity (activity_ref_po, activity_ref_user, activity_type, activity_notes, activity_create_date)
		VALUES (?, ?, ?, ?, NOW())`, poID, userID, actType, notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ActivityRepo) GetFirstByType(ctx context.Context, poID, actType string) (*models.Activity, error) {
	var v models.Activity
	err := r.db.QueryRowContext(ctx, `
		SELECT ac.activity_id, ac.activity_ref_po, ac.activity_ref_user, ac.activity_type, ac.activity_notes,
		       ac.activity_create_date, ac.activity_modify_date, COALESCE(ad.admin_name, ''), COALESCE(ad.admin_email, '')
		FROM T_Activity ac
		LEFT JOIN T_Admin ad ON ac.activity_ref_user = ad.admin_id
		WHERE ac.activity_ref_po = ? AND ac.activity_type = ?
		ORDER BY ac.activity_id ASC LIMIT 1`, poID, actType).
		Scan(&v.ID, &v.RefPO, &v.RefUser, &v.Type, &v.Notes, &v.CreateDate, &v.ModifyDate, &v.AdminName, &v.AdminImg)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
