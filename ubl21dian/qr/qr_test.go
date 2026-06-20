package qr

import "testing"

func TestURL(t *testing.T) {
	cases := []struct {
		env, want string
	}{
		{"1", "https://catalogo-vpfe.dian.gov.co/document/searchqr?documentkey=abc123"},
		{"2", "https://catalogo-vpfe-hab.dian.gov.co/document/searchqr?documentkey=abc123"},
	}
	for _, c := range cases {
		if got := URL(c.env, "abc123"); got != c.want {
			t.Errorf("URL(%q, ...) = %q, want %q", c.env, got, c.want)
		}
	}
}
