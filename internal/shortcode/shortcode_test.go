package shortcode_test

import (
	"strings"
	"testing"

	"url-shortener/internal/shortcode"
)

func TestEncode_LengthAndCharset(t *testing.T) {
	for _, id := range []uint64{0, 1, 42, 1000, 1 << 20, 1 << 40} {
		code := shortcode.Encode(id)
		if len(code) != shortcode.Length {
			t.Errorf("bad length for %d: got %d, want %d", id, len(code), shortcode.Length)
		}
		for _, r := range code {
			if !strings.ContainsRune(shortcode.Alphabet, r) {
				t.Errorf("%q has %q in it, not in alphabet", code, r)
			}
		}
	}
}

func TestEncode_Unique(t *testing.T) {
	const n = 200000

	seen := make(map[string]uint64, n)
	for id := uint64(0); id < n; id++ {
		code := shortcode.Encode(id)
		if prev, ok := seen[code]; ok {
			t.Fatalf("same code for %d and %d: %q", prev, id, code)
		}
		seen[code] = id
	}
}

func TestEncode_ZeroIsPadded(t *testing.T) {
	want := strings.Repeat(string(shortcode.Alphabet[0]), shortcode.Length)
	if got := shortcode.Encode(0); got != want {
		t.Errorf("zero code = %q, want %q", got, want)
	}
}
