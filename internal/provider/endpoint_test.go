package provider

import "testing"

func TestDeriveAPIBaseURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "https://api.snyk.io"},
		{"   ", "https://api.snyk.io"},
		{"api.snyk.io", "https://api.snyk.io"},
		{"api.snyk.io/", "https://api.snyk.io"},
		{"/api.snyk.io", "https://api.snyk.io"},
		{"https://api.snyk.io", "https://api.snyk.io"},
		{"https://api.snyk.io/", "https://api.snyk.io"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"HTTP://EXAMPLE.COM/path", "HTTP://EXAMPLE.COM/path"},
	}
	for _, tt := range tests {
		if got := deriveAPIBaseURL(tt.in); got != tt.want {
			t.Errorf("deriveAPIBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
