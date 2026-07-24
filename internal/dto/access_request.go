package dto

import (
	"errors"
	"strings"
)

// AccessRequest adalah payload create/update access (permission) individual.
type AccessRequest struct {
	Title  string `json:"title"`
	Module string `json:"module"`
	Slug   string `json:"slug"`
}

func (r *AccessRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Module = strings.TrimSpace(r.Module)
	r.Slug = strings.TrimSpace(strings.ToLower(r.Slug))
}

func (r AccessRequest) Validate() error {
	if r.Title == "" {
		return errors.New("judul akses wajib diisi")
	}
	if r.Module == "" {
		return errors.New("modul akses wajib diisi")
	}
	if r.Slug == "" {
		return errors.New("slug akses wajib diisi")
	}
	return nil
}