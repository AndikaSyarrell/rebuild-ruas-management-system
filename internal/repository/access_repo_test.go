package repository

// NOTE: butuh github.com/DATA-DOG/go-sqlmock (dependency eksternal, test-only).
// Sandbox tempat file ini ditulis tidak punya akses ke proxy.golang.org untuk
// mengunduhnya, sehingga belum sempat di-compile/dijalankan di lingkungan
// tersebut. Jalankan `go get github.com/DATA-DOG/go-sqlmock@latest` lalu
// `go test ./internal/repository/...` di mesin Anda untuk memverifikasi.

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func newMockAccessRepo(t *testing.T) (*AccessRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return NewAccessRepo(db), mock, func() { db.Close() }
}

func TestAccessRepo_HasAccess_Granted(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1
			  FROM T_Admin a
			  JOIN T_Role_Access ra ON ra.role_access_ref_role = a.admin_ref_role
			  JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
			  WHERE a.admin_id = ? AND ac.access_slug = ? AND a.admin_active = 'active'
			  LIMIT 1`)).
		WithArgs("ADM001", "edit_po").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	allowed, err := repo.HasAccess(context.Background(), "ADM001", "edit_po")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected HasAccess to return true when a matching row is found")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAccessRepo_HasAccess_Denied_NoRows(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1
			  FROM T_Admin a
			  JOIN T_Role_Access ra ON ra.role_access_ref_role = a.admin_ref_role
			  JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
			  WHERE a.admin_id = ? AND ac.access_slug = ? AND a.admin_active = 'active'
			  LIMIT 1`)).
		WithArgs("ADM002", "delete_po").
		WillReturnError(sql.ErrNoRows)

	allowed, err := repo.HasAccess(context.Background(), "ADM002", "delete_po")
	if err != nil {
		// sql.ErrNoRows HARUS diterjemahkan jadi (false, nil), BUKAN error -
		// "tidak punya akses" adalah hasil valid, bukan kegagalan query.
		t.Fatalf("expected sql.ErrNoRows to be treated as (false, nil), got error: %v", err)
	}
	if allowed {
		t.Error("expected HasAccess to return false when no matching row is found")
	}
}

func TestAccessRepo_HasAccess_QueryError_Propagates(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1
			  FROM T_Admin a
			  JOIN T_Role_Access ra ON ra.role_access_ref_role = a.admin_ref_role
			  JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
			  WHERE a.admin_id = ? AND ac.access_slug = ? AND a.admin_active = 'active'
			  LIMIT 1`)).
		WithArgs("ADM003", "edit_po").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.HasAccess(context.Background(), "ADM003", "edit_po")
	if err == nil {
		t.Error("expected genuine DB errors (not ErrNoRows) to propagate")
	}
}

func TestAccessRepo_ListPaged_ReturnsRowsAndTotal(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM T_Access`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT access_id, access_title, access_slug, access_module, access_sort, access_create_date, access_modify_date
		 FROM T_Access ORDER BY access_module ASC, access_create_date ASC LIMIT ? OFFSET ?`)).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows(
			[]string{"access_id", "access_title", "access_slug", "access_module", "access_sort", "access_create_date", "access_modify_date"},
		).
			AddRow(1, "Buat PO", "create_po", "po", 1, now, now).
			AddRow(2, "Ubah PO", "edit_po", "po", 2, now, now))

	list, total, err := repo.ListPaged(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}
	if list[0].Slug != "create_po" {
		t.Errorf("expected first row slug=create_po, got %s", list[0].Slug)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAccessRepo_GetAccessIDsForRole(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT role_access_ref_access FROM T_Role_Access WHERE role_access_ref_role = ?`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"role_access_ref_access"}).AddRow(3).AddRow(5).AddRow(9))

	ids, err := repo.GetAccessIDsForRole(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 access IDs, got %d", len(ids))
	}
	if ids[0] != 3 || ids[1] != 5 || ids[2] != 9 {
		t.Errorf("expected [3,5,9], got %v", ids)
	}
}

func TestAccessRepo_ReplaceRoleAccess_Transactional(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM T_Role_Access WHERE role_access_ref_role = ?`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectPrepare(regexp.QuoteMeta(
		`INSERT INTO T_Role_Access (role_access_ref_role, role_access_ref_access, role_access_create_date) VALUES (?, ?, NOW())`))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO T_Role_Access (role_access_ref_role, role_access_ref_access, role_access_create_date) VALUES (?, ?, NOW())`)).
		WithArgs(1, 10).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO T_Role_Access (role_access_ref_role, role_access_ref_access, role_access_create_date) VALUES (?, ?, NOW())`)).
		WithArgs(1, 11).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	err := repo.ReplaceRoleAccess(context.Background(), 1, []int{10, 11})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAccessRepo_ReplaceRoleAccess_RollsBackOnError(t *testing.T) {
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM T_Role_Access WHERE role_access_ref_role = ?`)).
		WithArgs(1).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.ReplaceRoleAccess(context.Background(), 1, []int{10})
	if err == nil {
		t.Error("expected error to propagate when DELETE fails mid-transaction")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations (rollback likely not called): %v", err)
	}
}

func TestAccessRepo_ReplaceRoleAccess_EmptySliceSkipsInsert(t *testing.T) {
	// Kalau accessIDs kosong (role dilucuti semua aksesnya), DELETE tetap
	// jalan tapi TIDAK boleh mencoba INSERT apapun.
	repo, mock, cleanup := newMockAccessRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM T_Role_Access WHERE role_access_ref_role = ?`)).
		WithArgs(2).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	err := repo.ReplaceRoleAccess(context.Background(), 2, []int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
