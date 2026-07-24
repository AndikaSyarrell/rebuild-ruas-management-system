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