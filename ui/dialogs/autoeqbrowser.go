package dialogs

import (
	"context"
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"github.com/deluan/sanitize"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/util"
)

// AutoEQBrowser allows users to browse and select AutoEQ headphone profiles
type AutoEQBrowser struct {
	SearchDialog      *SearchDialog
	manager           *backend.AutoEQManager
	allProfileResults []*mediaprovider.SearchResult
	OnProfileSelected func(*backend.AutoEQProfile)
}

func NewAutoEQBrowser(manager *backend.AutoEQManager, im util.ImageFetcher) *AutoEQBrowser {
	ab := &AutoEQBrowser{
		manager: manager,
	}
	sd := NewSearchDialog(
		im,
		lang.L("Browse Headphone Profiles"),
		lang.L("Cancel"),
		ab.onSearched,
	)
	sd.PlaceholderText = lang.L("Search headphones...")
	ab.SearchDialog = sd
	return ab
}

func (ab *AutoEQBrowser) fetchAllProfiles() error {
	ctx := context.Background()
	profiles, err := ab.manager.FetchIndex(ctx)
	if err != nil {
		log.Printf("Error fetching AutoEQ index: %v", err)
		// Show empty results on error
		ab.allProfileResults = []*mediaprovider.SearchResult{}
		return fmt.Errorf("failed to fetch AutoEQ index: %w", err)
	}

	log.Printf("Successfully fetched %d AutoEQ profiles", len(profiles))

	// Convert to SearchResult format for display
	ab.allProfileResults = sharedutil.MapSlice(profiles, ab.profileToSearchResult)
	return nil
}

func (ab *AutoEQBrowser) profileToSearchResult(profile backend.AutoEQProfileMetadata) *mediaprovider.SearchResult {
	// Format secondary text as "type · source" (e.g., "over-ear · oratory1990")
	subtitle := ""
	if profile.Type != "" {
		subtitle = profile.Type
	}
	if profile.Source != "" {
		if subtitle != "" {
			subtitle += " · "
		}
		subtitle += profile.Source
	}

	return &mediaprovider.SearchResult{
		Name:       profile.Name,
		ID:         profile.Path, // Store path as ID for retrieval
		Type:       mediaprovider.ContentTypeAlbum, // Use album icon (looks like headphones)
		ArtistName: subtitle,
		Size:       0, // Don't show track count
	}
}

func (ab *AutoEQBrowser) onSearched(query string) []*mediaprovider.SearchResult {
	if ab.allProfileResults == nil {
		if err := ab.fetchAllProfiles(); err != nil {
			log.Printf("Failed to load AutoEQ profiles: %v", err)
			// Return a single error result
			return []*mediaprovider.SearchResult{
				{
					Name:       lang.L("Error loading AutoEQ profiles"),
					ArtistName: lang.L("Check network connection and try again"),
					Type:       mediaprovider.ContentTypePlaylist,
				},
			}
		}
	}

	if query == "" {
		return ab.allProfileResults
	}

	// Filter by name (case-insensitive, accent-insensitive)
	return sharedutil.FilterSlice(ab.allProfileResults, func(result *mediaprovider.SearchResult) bool {
		return strings.Contains(
			sanitize.Accents(strings.ToLower(result.Name)),
			sanitize.Accents(strings.ToLower(query)),
		)
	})
}

func (ab *AutoEQBrowser) SetOnDismiss(onDismiss func()) {
	ab.SearchDialog.OnDismiss = onDismiss
}

func (ab *AutoEQBrowser) SetOnProfileSelected(callback func(*backend.AutoEQProfile)) {
	ab.OnProfileSelected = callback
	ab.SearchDialog.OnNavigateTo = func(_ mediaprovider.ContentType, profilePath string) {
		// Fetch the full profile data
		ctx := context.Background()
		profile, err := ab.manager.FetchProfile(ctx, profilePath)
		if err != nil {
			log.Printf("Error fetching AutoEQ profile: %v", err)
			// TODO: Show error dialog to user
			return
		}

		if ab.OnProfileSelected != nil {
			ab.OnProfileSelected(profile)
		}
	}
}

func (ab *AutoEQBrowser) MinSize() fyne.Size {
	return ab.SearchDialog.MinSize()
}

func (ab *AutoEQBrowser) GetSearchEntry() fyne.Focusable {
	return ab.SearchDialog.GetSearchEntry()
}

func (ab *AutoEQBrowser) Show() {
	ab.SearchDialog.Show()
}

func (ab *AutoEQBrowser) Hide() {
	ab.SearchDialog.Hide()
}

func (ab *AutoEQBrowser) Refresh() {
	ab.SearchDialog.Refresh()
}

// ShowErrorDialog displays an error message to the user
func ShowAutoEQError(window fyne.Window, err error) {
	title := lang.L("Error")
	message := lang.L("Failed to load profile")

	if err == backend.ErrProfileNotFound {
		message = lang.L("Profile not found")
	} else if strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "connection") {
		message = lang.L("Network error. Check connection.")
	}

	// Use a simple dialog (would need to import "fyne.io/fyne/v2/dialog")
	// For now just log it
	log.Printf("AutoEQ Error: %s - %v", message, err)
	fmt.Printf("%s: %s\n", title, message)
}
