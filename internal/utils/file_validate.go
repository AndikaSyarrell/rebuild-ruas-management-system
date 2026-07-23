package utils

import (
	"errors"
	"io"
	"net/http"
)

var ErrFileTypeMismatch = errors.New("isi file tidak sesuai dengan format yang diklaim")

const sniffLen = 512

// DetectedContentType membaca beberapa byte pertama dari sebuah io.ReadSeeker
// untuk mendeteksi MIME type sesungguhnya (magic number), lalu me-reset
// posisi baca ke awal supaya file masih bisa dibaca utuh oleh pemanggil
// berikutnya (io.Copy ke disk, dsb).
func DetectedContentType(f io.ReadSeeker) (string, error) {
	buf := make([]byte, sniffLen)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

// ValidateFileContent memastikan isi file (dideteksi via magic number) benar-benar
// termasuk salah satu MIME type yang diizinkan - BUKAN sekadar mempercayai
// ekstensi nama file yang gampang dipalsukan (rename .exe jadi .jpg tetap
// lolos kalau hanya cek strings.HasSuffix / filepath.Ext).
func ValidateFileContent(f io.ReadSeeker, allowedMIME map[string]bool) error {
	detected, err := DetectedContentType(f)
	if err != nil {
		return err
	}
	if !allowedMIME[detected] {
		return ErrFileTypeMismatch
	}
	return nil
}