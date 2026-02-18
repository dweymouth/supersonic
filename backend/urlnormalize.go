package backend

import "strings"

// NormalizeServerURL applies common normalization to a server URL:
// prepends "http://" if no scheme is present, then strips trailing slashes.
func NormalizeServerURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	rawURL = strings.TrimRight(rawURL, "/")
	return rawURL
}

// NormalizeJellyfinURL applies common normalization then additionally strips
// known Jellyfin web UI path suffixes (/web/index.html and /web).
func NormalizeJellyfinURL(rawURL string) string {
	rawURL = NormalizeServerURL(rawURL)
	if strings.HasSuffix(rawURL, "/web/index.html") {
		rawURL = strings.TrimSuffix(rawURL, "/web/index.html")
	} else if strings.HasSuffix(rawURL, "/web") {
		rawURL = strings.TrimSuffix(rawURL, "/web")
	}
	return rawURL
}
