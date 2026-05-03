package common

import "testing"

func TestNormalizeEmailDomain(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "lowercases and trims", in: " User@PSOVV.COM ", want: "psovv.com", ok: true},
		{name: "trims trailing dot", in: "user@example.com.", want: "example.com", ok: true},
		{name: "missing domain", in: "user@", ok: false},
		{name: "missing at", in: "user", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeEmailDomain(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("NormalizeEmailDomain(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestIsDisposableEmailDomain(t *testing.T) {
	if !IsDisposableEmailDomain("psovv.com") {
		t.Fatal("expected listed domain to be disposable")
	}
	if !IsDisposableEmailDomain("mail.xghff.com") {
		t.Fatal("expected listed subdomain to be disposable")
	}
	if IsDisposableEmailDomain("example.com") {
		t.Fatal("did not expect normal domain to be disposable")
	}
}
