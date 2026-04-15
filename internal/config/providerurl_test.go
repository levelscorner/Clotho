package config

import (
	"strings"
	"testing"
)

func TestValidateProviderURL(t *testing.T) {
	cases := []struct {
		name         string
		url          string
		extra        []string
		wantErr      bool
		errSubstring string
	}{
		{name: "empty passes", url: "", wantErr: false},
		{name: "localhost:11434 ok", url: "http://localhost:11434", wantErr: false},
		{name: "127.0.0.1 ok", url: "http://127.0.0.1:8188", wantErr: false},
		{name: "host.docker.internal ok", url: "http://host.docker.internal:8880", wantErr: false},
		{name: "0.0.0.0 rejected", url: "http://0.0.0.0:1234", wantErr: true, errSubstring: "0.0.0.0"},
		{name: "external host rejected", url: "http://example.com", wantErr: true, errSubstring: "allowlist"},
		{name: "public IP rejected", url: "http://8.8.8.8:80", wantErr: true, errSubstring: "allowlist"},
		{name: "ftp scheme rejected", url: "ftp://localhost:21", wantErr: true, errSubstring: "scheme"},
		{name: "https ok", url: "https://localhost:11434", wantErr: false},
		{name: "extra hosts accepted", url: "http://gpu-rack.lan:11434", extra: []string{"gpu-rack.lan"}, wantErr: false},
		{name: "extra hosts with port", url: "http://gpu:11434", extra: []string{"gpu:11434"}, wantErr: false},
		{name: "malformed URL rejected", url: "http://[::bogus", wantErr: true, errSubstring: "invalid URL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProviderURL("TEST", tc.url, tc.extra)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.url, err)
			}
			if tc.errSubstring != "" && err != nil && !strings.Contains(err.Error(), tc.errSubstring) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstring)
			}
		})
	}
}

func TestParseAllowedHosts(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{in: "", want: nil},
		{in: "  ", want: nil},
		{in: "a,b,c", want: []string{"a", "b", "c"}},
		{in: " a , b ,, c ", want: []string{"a", "b", "c"}},
		{in: "gpu-rack.lan:11434", want: []string{"gpu-rack.lan:11434"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := ParseAllowedHosts(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("i=%d: got %q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
