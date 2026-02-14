package mpd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

const (
	deezerBaseURL      = "https://api.deezer.com"
	artistInfoTimeout  = 10 * time.Second
	artistInfoCacheTTL = 24 * time.Hour
)

// artistInfoCacheEntry stores cached artist info with expiration.
type artistInfoCacheEntry struct {
	info      *mediaprovider.ArtistInfo
	expiresAt time.Time
}

// artistInfoFetcher fetches artist info from Deezer (images) and Wikipedia (biography).
type artistInfoFetcher struct {
	httpClient *http.Client
	cache      map[string]artistInfoCacheEntry
	cacheMu    sync.RWMutex
	language   string // User's preferred language code (e.g., "it", "de", "fr")
}

func newArtistInfoFetcher(language string) *artistInfoFetcher {
	return &artistInfoFetcher{
		httpClient: &http.Client{
			Timeout: artistInfoTimeout,
		},
		cache:    make(map[string]artistInfoCacheEntry),
		language: language,
	}
}

// clearCache clears the artist info cache.
func (f *artistInfoFetcher) clearCache() {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()
	f.cache = make(map[string]artistInfoCacheEntry)
}

// fetchArtistInfo fetches artist info from Deezer (images) and Wikipedia (biography) with caching.
func (f *artistInfoFetcher) fetchArtistInfo(artistName string) (*mediaprovider.ArtistInfo, error) {
	if artistName == "" {
		return &mediaprovider.ArtistInfo{}, nil
	}

	// Normalize artist name for cache key
	cacheKey := strings.ToLower(artistName)

	// Check cache first
	f.cacheMu.RLock()
	if entry, ok := f.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		f.cacheMu.RUnlock()
		return entry.info, nil
	}
	f.cacheMu.RUnlock()

	// Fetch image from Deezer
	info, err := f.fetchFromDeezer(artistName)
	if err != nil {
		// Start with empty result if Deezer fails
		info = &mediaprovider.ArtistInfo{}
	}

	// Fetch biography from Wikipedia
	biography, wikiURL := f.fetchBiographyFromWikipedia(artistName)
	if biography != "" {
		info.Biography = biography
		// If we got a Wikipedia URL and don't have a link yet, use it
		if wikiURL != "" && info.LastFMUrl == "" {
			info.LastFMUrl = wikiURL
		}
	}

	// Store in cache
	f.cacheMu.Lock()
	f.cache[cacheKey] = artistInfoCacheEntry{
		info:      info,
		expiresAt: time.Now().Add(artistInfoCacheTTL),
	}
	f.cacheMu.Unlock()

	return info, nil
}

// fetchFromDeezer fetches artist info from Deezer API.
func (f *artistInfoFetcher) fetchFromDeezer(artistName string) (*mediaprovider.ArtistInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), artistInfoTimeout)
	defer cancel()

	// Build URL for Deezer artist search
	reqURL := fmt.Sprintf("%s/search/artist?q=%s", deezerBaseURL, url.QueryEscape(artistName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Supersonic/1.0 (https://github.com/dweymouth/supersonic)")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artist info fetch failed with status %d", resp.StatusCode)
	}

	var result deezerSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if result.Error.Code != 0 {
		return nil, fmt.Errorf("deezer API error %d: %s", result.Error.Code, result.Error.Message)
	}

	if len(result.Data) == 0 {
		return &mediaprovider.ArtistInfo{}, nil
	}

	// Find best match (exact name match preferred)
	var bestMatch *deezerArtist
	artistLower := strings.ToLower(artistName)
	for i := range result.Data {
		if strings.ToLower(result.Data[i].Name) == artistLower {
			bestMatch = &result.Data[i]
			break
		}
	}
	if bestMatch == nil {
		bestMatch = &result.Data[0]
	}

	info := &mediaprovider.ArtistInfo{}

	// Prefer XL image, fall back to big, then medium
	if bestMatch.PictureXL != "" && !isDeezerPlaceholder(bestMatch.PictureXL) {
		info.ImageURL = bestMatch.PictureXL
	} else if bestMatch.PictureBig != "" && !isDeezerPlaceholder(bestMatch.PictureBig) {
		info.ImageURL = bestMatch.PictureBig
	} else if bestMatch.PictureMedium != "" && !isDeezerPlaceholder(bestMatch.PictureMedium) {
		info.ImageURL = bestMatch.PictureMedium
	}

	// Set link to Deezer artist page
	if bestMatch.Link != "" {
		info.LastFMUrl = bestMatch.Link
	}

	return info, nil
}

// isDeezerPlaceholder checks if the URL is a Deezer default placeholder image.
func isDeezerPlaceholder(url string) bool {
	// Deezer placeholder images contain "d-artist" or specific placeholder patterns
	return strings.Contains(url, "/artist//") || strings.Contains(url, "d-artist")
}

// fetchBiographyFromWikipedia fetches artist biography from Wikipedia.
// It tries the user's preferred language first, then always falls back to English.
func (f *artistInfoFetcher) fetchBiographyFromWikipedia(artistName string) (string, string) {
	// Determine which languages to try - always include English as fallback
	langs := []string{"en"}

	// Get effective language - resolve "auto" to system language
	lang := f.language
	if lang == "" || lang == "auto" {
		lang = getSystemLanguage()
	}

	// If we have a non-English language, try it first
	if lang != "" && lang != "en" {
		wikiLang := mapToWikipediaLang(lang)
		if wikiLang != "" && wikiLang != "en" {
			langs = []string{wikiLang, "en"}
		}
	}

	// Try each language until we get a result
	for _, lang := range langs {
		extract, pageURL := f.fetchWikipediaBio(artistName, lang)
		if extract != "" {
			return extract, pageURL
		}
	}

	return "", ""
}

// getSystemLanguage detects the system language from environment variables.
func getSystemLanguage() string {
	for _, envVar := range []string{"LANG", "LC_MESSAGES", "LC_ALL", "LANGUAGE"} {
		if val := os.Getenv(envVar); val != "" {
			// Extract language code from locale (e.g., "it_IT.UTF-8" -> "it")
			lang := strings.Split(val, "_")[0]
			lang = strings.Split(lang, ".")[0]
			if lang != "" && lang != "C" && lang != "POSIX" {
				return lang
			}
		}
	}
	return ""
}

// mapToWikipediaLang maps app language codes to Wikipedia language codes.
func mapToWikipediaLang(appLang string) string {
	// Map special cases
	switch appLang {
	case "zhHans", "zhHant", "zh":
		return "zh"
	case "pt_BR":
		return "pt"
	default:
		// Most language codes match directly (de, fr, it, es, etc.)
		return appLang
	}
}

// fetchWikipediaBio fetches biography from a specific Wikipedia language edition.
func (f *artistInfoFetcher) fetchWikipediaBio(artistName, lang string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), artistInfoTimeout)
	defer cancel()

	// Build URL for the specific language Wikipedia
	// Format: https://{lang}.wikipedia.org/api/rest_v1/page/summary/{title}
	title := strings.ReplaceAll(artistName, " ", "_")
	reqURL := fmt.Sprintf("https://%s.wikipedia.org/api/rest_v1/page/summary/%s", lang, url.PathEscape(title))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Supersonic/1.0 (https://github.com/dweymouth/supersonic)")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var result wikipediaSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", ""
	}

	// Return the extract (biography) and Wikipedia page URL
	pageURL := ""
	if result.ContentURLs.Desktop.Page != "" {
		pageURL = result.ContentURLs.Desktop.Page
	}

	return result.Extract, pageURL
}

// Deezer API response types

type deezerSearchResponse struct {
	Data  []deezerArtist `json:"data"`
	Error deezerError    `json:"error"`
}

type deezerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type deezerArtist struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Link          string `json:"link"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXL     string `json:"picture_xl"`
	NbFan         int    `json:"nb_fan"`
}

// Wikipedia API response types

type wikipediaSummary struct {
	Title       string `json:"title"`
	Extract     string `json:"extract"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}
