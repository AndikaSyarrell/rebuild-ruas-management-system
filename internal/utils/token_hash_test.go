package utils

import "testing"

func TestHashToken_Deterministic(t *testing.T) {
	token := "DEMO-ACTIVATION-TOKEN-0001"
	h1 := HashToken(token)
	h2 := HashToken(token)
	if h1 != h2 {
		t.Errorf("expected deterministic hash, got %s vs %s", h1, h2)
	}
}

func TestHashToken_DifferentInputsDifferentHashes(t *testing.T) {
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if h1 == h2 {
		t.Error("expected different tokens to produce different hashes")
	}
}

func TestHashToken_Length(t *testing.T) {
	// SHA-256 hex digest harus selalu 64 karakter, berapapun panjang input.
	h := HashToken("x")
	if len(h) != 64 {
		t.Errorf("expected 64-char hex digest, got %d chars", len(h))
	}
}

func TestHashToken_NeverEqualsRawInput(t *testing.T) {
	token := "plaintext-should-not-survive"
	h := HashToken(token)
	if h == token {
		t.Error("hash must never equal the raw input token")
	}
}

func TestHashToken_EmptyString(t *testing.T) {
	// Tidak boleh panic pada input kosong (defensif; walau di alur nyata
	// token kosong seharusnya sudah ditolak di layer lain sebelum sampai sini).
	h := HashToken("")
	if len(h) != 64 {
		t.Errorf("expected 64-char hex digest even for empty input, got %d", len(h))
	}
}

func TestHashToken_KnownVector(t *testing.T) {
	// Known-answer test terhadap SHA-256("abc") - vektor uji standar (NIST),
	// memastikan implementasi tidak diam-diam berubah algoritma di masa depan.
	got := HashToken("abc")
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Errorf("expected SHA-256(\"abc\") = %s, got %s", want, got)
	}
}
