package dto

import (
	"errors"
	"strings"
)

// LoginRequest adalah payload login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r LoginRequest) Validate() error {
	if r.Email == "" || r.Password == "" {
		return errors.New("email dan password wajib diisi")
	}
	return nil
}

// RefreshRequest dipakai untuk refresh token maupun logout (refresh_token
// opsional saat logout).
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refresh_token wajib diisi")
	}
	return nil
}

// ResetPasswordRequest - payload ganti password (user sudah login).
type ResetPasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (r ResetPasswordRequest) Validate() error {
	if r.OldPassword == "" || r.NewPassword == "" || r.ConfirmPassword == "" {
		return errors.New("semua field wajib diisi")
	}
	if r.NewPassword != r.ConfirmPassword {
		return errors.New("konfirmasi password baru tidak cocok")
	}
	return nil
}

// ForgotPasswordRequest - payload permintaan reset password lewat email.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func (r *ForgotPasswordRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r ForgotPasswordRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email wajib diisi")
	}
	return nil
}

// ResetWithCodeRequest - payload reset password memakai kode dari email.
type ResetWithCodeRequest struct {
	ResetCode   string `json:"reset_code"`
	NewPassword string `json:"new_password"`
}

func (r ResetWithCodeRequest) Validate() error {
	if r.ResetCode == "" || r.NewPassword == "" {
		return errors.New("data tidak lengkap")
	}
	return nil
}

// ActivateRequest - payload aktivasi akun admin baru.
type ActivateRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (r *ActivateRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r ActivateRequest) Validate() error {
	if r.Email == "" || r.Token == "" || r.Password == "" {
		return errors.New("data tidak lengkap")
	}
	return nil
}