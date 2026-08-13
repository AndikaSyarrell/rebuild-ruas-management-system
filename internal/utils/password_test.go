package utils

// CATATAN: file ini butuh golang.org/x/crypto/bcrypt (dependency eksternal).
// Sandbox tempat file ini dibuat TIDAK punya akses ke proxy.golang.org,
// sehingga test ini belum bisa dijalankan/di-compile di lingkungan tersebut.
// Jalankan `go test ./internal/utils/...` di mesin Anda sendiri (dengan
// akses internet normal) untuk memverifikasi.

import "testing"

func TestHashPassword_ProducesDifferentHashForSameInput(t *testing.T) {
	// bcrypt menyertakan salt acak - dua kali hash password yang SAMA harus
	// menghasilkan string hash yang BEDA (bukan berarti gagal, ini bcrypt bekerja benar).
	h1, err := HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("expected different bcrypt hashes for same password due to random salt")
	}
}

func TestHashPassword_NeverEqualsPlainInput(t *testing.T) {
	h, err := HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == "Admin123!" {
		t.Error("hash must never equal the plaintext password")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	h, err := HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !CheckPassword(h, "Admin123!") {
		t.Error("expected CheckPassword to return true for matching password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	h, err := HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if CheckPassword(h, "WrongPassword!") {
		t.Error("expected CheckPassword to return false for non-matching password")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	// Guard eksplisit di kode: hash kosong (admin.Password nil di-dereference
	// jadi "" oleh pemanggil) harus selalu gagal, bukan panic ataupun match.
	if CheckPassword("", "AnyPassword123!") {
		t.Error("expected CheckPassword to return false when hash is empty")
	}
}

func TestCheckPassword_MalformedHash(t *testing.T) {
	if CheckPassword("not-a-real-bcrypt-hash", "Admin123!") {
		t.Error("expected CheckPassword to return false for malformed hash")
	}
}

func TestValidatePasswordStrength_ValidPassword(t *testing.T) {
	if err := ValidatePasswordStrength("Admin123!"); err != nil {
		t.Errorf("expected valid password to pass, got error: %v", err)
	}
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	err := ValidatePasswordStrength("Ab1")
	if err != ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword for short password, got %v", err)
	}
}

func TestValidatePasswordStrength_NoUppercase(t *testing.T) {
	err := ValidatePasswordStrength("admin123!")
	if err != ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword when missing uppercase, got %v", err)
	}
}

func TestValidatePasswordStrength_NoLowercase(t *testing.T) {
	err := ValidatePasswordStrength("ADMIN123!")
	if err != ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword when missing lowercase, got %v", err)
	}
}

func TestValidatePasswordStrength_NoDigit(t *testing.T) {
	err := ValidatePasswordStrength("AdminPassword!")
	if err != ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword when missing digit, got %v", err)
	}
}

func TestValidatePasswordStrength_SymbolsNotRequired(t *testing.T) {
	// Simbol tidak diwajibkan (sesuai komentar asli di password.go) - kombinasi
	// upper+lower+digit dengan panjang cukup harus tetap lolos tanpa simbol.
	if err := ValidatePasswordStrength("AdminPassword123"); err != nil {
		t.Errorf("expected password without symbols to pass, got error: %v", err)
	}
}

func TestValidatePasswordStrength_ExactlyEightChars(t *testing.T) {
	// Batas panjang minimum persis 8 karakter - kasus tepi (boundary).
	if err := ValidatePasswordStrength("Abcdef1!"); err != nil {
		t.Errorf("expected exactly-8-char valid password to pass, got error: %v", err)
	}
}

func TestValidatePasswordStrength_SevenCharsFails(t *testing.T) {
	err := ValidatePasswordStrength("Abcdef1")
	if err != ErrWeakPassword {
		t.Errorf("expected 7-char password to fail length check, got %v", err)
	}
}
