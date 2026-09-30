package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeQuotationNo(t *testing.T) {
	cases := map[string]string{"  quot-001 ": "QUOT-001", "QUOT-001": "QUOT-001", "   ": "", "": ""}
	for in, want := range cases {
		if got := NormalizeQuotationNo(in); got != want {
			t.Errorf("NormalizeQuotationNo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHasCompletedMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPRQuotationRepo(db)
	q := regexp.QuoteMeta(`SELECT 1 FROM T_Purchase_Request WHERE pr_ref_quotation = ? AND pr_status = 'completed' LIMIT 1`)

	mock.ExpectQuery(q).WithArgs(7).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	if ok, err := repo.HasCompletedMember(context.Background(), 7); err != nil || !ok {
		t.Fatalf("expected (true,nil), got (%v,%v)", ok, err)
	}

	mock.ExpectQuery(q).WithArgs(8).WillReturnError(sql.ErrNoRows)
	if ok, err := repo.HasCompletedMember(context.Background(), 8); err != nil || ok {
		t.Fatalf("expected (false,nil), got (%v,%v)", ok, err)
	}
}

func TestLinkToPO_ConflictRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE T_Pr_Quotation")).
		WithArgs("PO1", "ADM1", 7).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 baris = keburu di-link pihak lain
	mock.ExpectRollback()

	err = NewPRQuotationRepo(db).LinkToPO(context.Background(), 7, "PO1", "ORD/1", "ADM1")
	if !errors.Is(err, ErrQuotationLinkConflict) {
		t.Fatalf("expected ErrQuotationLinkConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestLinkToPO_SyncsMembersAndCommits(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE T_Pr_Quotation")).
		WithArgs("PO1", "ADM1", 7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE T_Purchase_Request SET pr_po_no = ?")).
		WithArgs("ORD/1", 7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	if err := NewPRQuotationRepo(db).LinkToPO(context.Background(), 7, "PO1", "ORD/1", "ADM1"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}