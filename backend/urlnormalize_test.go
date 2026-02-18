package backend

import "testing"

func TestNormalizeServerURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"http://192.168.1.1:8096", "http://192.168.1.1:8096"},
		{"https://music.example.com", "https://music.example.com"},
		{"192.168.1.1:4533", "http://192.168.1.1:4533"},
		{"music.example.com", "http://music.example.com"},
		{"http://192.168.1.1:8096/", "http://192.168.1.1:8096"},
		{"http://192.168.1.1:8096///", "http://192.168.1.1:8096"},
		{"192.168.1.1:8096/", "http://192.168.1.1:8096"},
	}
	for _, tt := range tests {
		got := NormalizeServerURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeServerURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeJellyfinURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"http://192.168.1.1:8096", "http://192.168.1.1:8096"},
		{"192.168.1.1:8096", "http://192.168.1.1:8096"},
		{"192.168.1.1:8096/", "http://192.168.1.1:8096"},
		{"192.168.1.1:8096/web/index.html", "http://192.168.1.1:8096"},
		{"http://192.168.1.1:8096/web/index.html", "http://192.168.1.1:8096"},
		{"http://192.168.1.1:8096/web/", "http://192.168.1.1:8096"},
		{"http://192.168.1.1:8096/web", "http://192.168.1.1:8096"},
		{"https://jellyfin.example.com/web/index.html", "https://jellyfin.example.com"},
		{"https://jellyfin.example.com/web/", "https://jellyfin.example.com"},
	}
	for _, tt := range tests {
		got := NormalizeJellyfinURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeJellyfinURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
