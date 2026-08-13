package utils

import (
	"bytes"
	"strings"
	"testing"
)

// fakeReadSeeker membungkus bytes.Reader supaya implement io.ReadSeeker,
// dipakai untuk mensimulasikan multipart.File di test tanpa upload sungguhan.
func newFakeFile(content []byte) *bytes.Reader {
	return bytes.NewReader(content)
}

func TestDetectedContentType_PNG(t *testing.T) {
	// 8 byte magic number PNG asli.
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	f := newFakeFile(pngHeader)

	ct, err := DetectedContentType(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("expected image/png, got %s", ct)
	}
}

func TestDetectedContentType_ResetsReadPosition(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xFF, 0xFF}
	f := newFakeFile(pngHeader)

	if _, err := DetectedContentType(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Setelah DetectedContentType, posisi baca HARUS kembali ke awal supaya
	// pemanggil berikutnya (io.Copy ke disk) tetap dapat file utuh.
	pos, err := f.Seek(0, 1) // io.SeekCurrent
	if err != nil {
		t.Fatalf("unexpected seek error: %v", err)
	}
	if pos != 0 {
		t.Errorf("expected read position reset to 0, got %d", pos)
	}
}

func TestDetectedContentType_PlainText(t *testing.T) {
	f := newFakeFile([]byte("hello world, this is plain text"))
	ct, err := DetectedContentType(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}
}

func TestValidateFileContent_Allowed(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	f := newFakeFile(pngHeader)

	allowed := map[string]bool{"image/png": true, "image/jpeg": true}
	if err := ValidateFileContent(f, allowed); err != nil {
		t.Errorf("expected file to pass validation, got error: %v", err)
	}
}

func TestValidateFileContent_Rejected(t *testing.T) {
	// Konten teks biasa yang di-rename seolah gambar - harus ditolak karena
	// magic number-nya tidak cocok whitelist, terlepas dari ekstensi nama file.
	f := newFakeFile([]byte("MZ this is actually an executable disguised as .jpg"))

	allowed := map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true}
	err := ValidateFileContent(f, allowed)
	if err == nil {
		t.Fatal("expected validation to reject mismatched content type")
	}
	if err != ErrFileTypeMismatch {
		t.Errorf("expected ErrFileTypeMismatch, got %v", err)
	}
}

func TestValidateFileContent_PDF(t *testing.T) {
	pdfHeader := []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	f := newFakeFile(pdfHeader)

	allowed := map[string]bool{"application/pdf": true}
	if err := ValidateFileContent(f, allowed); err != nil {
		t.Errorf("expected PDF to pass validation, got error: %v", err)
	}
}

func TestValidateFileContent_EmptyAllowlist(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	f := newFakeFile(pngHeader)

	err := ValidateFileContent(f, map[string]bool{})
	if err == nil {
		t.Error("expected rejection when allowlist is empty")
	}
}
