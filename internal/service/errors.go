package service

import "errors"

var (
	ErrAccountInactive     = errors.New("akun tidak aktif")
	ErrWrongOldPassword    = errors.New("password lama salah")
	ErrSamePassword        = errors.New("password baru sama dengan password lama")
	ErrInvalidCredentials  = errors.New("email atau password salah")
	ErrNotFound            = errors.New("data tidak ditemukan")
	ErrDuplicateEmail      = errors.New("email sudah terdaftar")
	ErrDuplicateOrderNum   = errors.New("nomor purchase order sudah terdaftar")
	ErrInvalidRefreshToken = errors.New("refresh token tidak valid")
)
