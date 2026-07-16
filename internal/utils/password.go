package utils

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var ErrWeakPassword = errors.New("password terlalu lemah")

// bcryptCost 12 adalah keseimbangan aman antara kekuatan hash dan latensi
// (di atas default 10). Naikkan seiring pertumbuhan kapasitas server.
const bcryptCost = 12

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword membandingkan password plain dengan hash secara constant-time
// (bcrypt.CompareHashAndPassword sudah constant-time by design).
func CheckPassword(hash, plain string) bool {
	if hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

// ValidatePasswordStrength menerapkan aturan minimal untuk password baru:
// panjang >= 8 karakter, serta kombinasi huruf besar, huruf kecil, dan angka.
// Simbol tidak diwajibkan supaya tidak terlalu menyulitkan, tapi boleh dipakai.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}

	return nil
}
