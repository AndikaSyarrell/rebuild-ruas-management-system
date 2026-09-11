package dto

import (
	"errors"
	"strings"
)

// TitleRequest adalah payload generik untuk modul lookup sederhana yang
// cuma punya satu field "title" (Region, Division, Unit). Dipakai bersama
// supaya tidak menduplikasi struct identik 3x.
type TitleRequest struct {
	Title string `json:"title"`
}

func (r *TitleRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
}

func (r TitleRequest) Validate() error {
	if r.Title == "" {
		return errors.New("judul wajib diisi")
	}
	return nil
}

type DivisionRequest struct {
	Title string `json:"title"`
	Code  string `json:"code"`
}

func (r *DivisionRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
}

func (r DivisionRequest) Validate() error {
	if r.Title == "" {
		return errors.New("judul divisi wajib diisi")
	}
	if r.Code == "" {
		return errors.New("kode divisi wajib diisi")
	}
	return nil
}