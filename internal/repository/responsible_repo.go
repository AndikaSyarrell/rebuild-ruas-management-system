package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
)

type ResponsibleRepo struct{ db db.Querier }

func NewResponsibleRepo(db db.Querier) *ResponsibleRepo { return &ResponsibleRepo{db: db} }

const responsibleSelect = `
	SELECT responsible_id, responsible_name, responsible_coa_code, responsible_status,
	       responsible_create_date, responsible_modify_date
	FROM T_Responsible`

func scanResponsible(row interface {
	Scan(dest ...interface{}) error
}) (*models.Responsible, error) {
	var v models.Responsible
	err := row.Scan(&v.ID, &v.Name, &v.CoaCode, &v.Status, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ResponsibleRepo) CoaCodeExists(ctx context.Context, coaCode string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM T_Responsible WHERE responsible_coa_code = ? LIMIT 1`, coaCode).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListSelect mengembalikan responsible berstatus AKTIF saja - dipakai untuk
// dropdown pr_ref_responsible saat create/update PR.
func (r *ResponsibleRepo) ListSelect(ctx context.Context) ([]models.Responsible, error) {
	rows, err := r.db.QueryContext(ctx, responsibleSelect+` WHERE responsible_status = 'active' ORDER BY responsible_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Responsible
	for rows.Next() {
		v, err := scanResponsible(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *ResponsibleRepo) ListPaged(ctx context.Context, page, perPage int, keyword string) ([]models.Responsible, int, error) {
	whereKeyword := keyword == ""

	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Responsible
		 WHERE (? OR responsible_name LIKE CONCAT('%', ?, '%') OR responsible_coa_code LIKE CONCAT('%', ?, '%'))`,
		whereKeyword, keyword, keyword).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		responsibleSelect+`
		 WHERE (? OR responsible_name LIKE CONCAT('%', ?, '%') OR responsible_coa_code LIKE CONCAT('%', ?, '%'))
		 ORDER BY responsible_create_date DESC LIMIT ? OFFSET ?`,
		whereKeyword, keyword, keyword, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Responsible
	for rows.Next() {
		v, err := scanResponsible(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

func (r *ResponsibleRepo) GetByID(ctx context.Context, id int) (*models.Responsible, error) {
	row := r.db.QueryRowContext(ctx, responsibleSelect+` WHERE responsible_id = ?`, id)
	return scanResponsible(row)
}

func (r *ResponsibleRepo) Create(ctx context.Context, name, coaCode, status string) (int64, error) {
	if status == "" {
		status = "active"
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Responsible (responsible_name, responsible_coa_code, responsible_status, responsible_create_date)
		 VALUES (?, ?, ?, NOW())`, name, coaCode, status)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ResponsibleRepo) Update(ctx context.Context, id int, name, coaCode string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Responsible SET responsible_name = ?, responsible_coa_code = ? WHERE responsible_id = ?`,
		name, coaCode, id)
	return err
}

func (r *ResponsibleRepo) ChangeStatus(ctx context.Context, id int, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Responsible SET responsible_status = ? WHERE responsible_id = ?`, status, id)
	return err
}

// Delete akan gagal (FK constraint fk_pr_responsible, ON DELETE RESTRICT)
// bila responsible ini masih dipakai oleh baris T_Purchase_Request manapun -
// itu perilaku yang diinginkan, bukan bug.
func (r *ResponsibleRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Responsible WHERE responsible_id = ?`, id)
	return err
}