package mediaprovider

import "time"

// Bit field flag for the ReleaseTypes property
type ReleaseType = int32

// Bit field of release types
type ReleaseTypes = int32

// Set of possible release types
// Taken from Picard:
//
//	(a) https://picard-docs.musicbrainz.org/en/config/options_releases.html
//	(b) https://musicbrainz.org/doc/Release_Group/Type
const (
	ReleaseTypeAlbum          ReleaseType = 0x0001
	ReleaseTypeAudiobook      ReleaseType = 0x0002
	ReleaseTypeAudioDrama     ReleaseType = 0x0004
	ReleaseTypeBroadcast      ReleaseType = 0x0008
	ReleaseTypeCompilation    ReleaseType = 0x0010
	ReleaseTypeDemo           ReleaseType = 0x0020
	ReleaseTypeDJMix          ReleaseType = 0x0040
	ReleaseTypeEP             ReleaseType = 0x0080
	ReleaseTypeFieldRecording ReleaseType = 0x0100
	ReleaseTypeInterview      ReleaseType = 0x0200
	ReleaseTypeLive           ReleaseType = 0x0400
	ReleaseTypeMixtape        ReleaseType = 0x0800
	ReleaseTypeRemix          ReleaseType = 0x1000
	ReleaseTypeSingle         ReleaseType = 0x2000
	ReleaseTypeSoundtrack     ReleaseType = 0x4000
	ReleaseTypeSpokenWord     ReleaseType = 0x8000
)

type Library struct {
	ID   string
	Name string
}

type ItemDate struct {
	Year  *int
	Month *int
	Day   *int
}

func (i ItemDate) After(other ItemDate) bool {
	a := []*int{i.Year, i.Month, i.Day}
	b := []*int{other.Year, other.Month, other.Day}
	for i := range a {
		if a[i] == nil {
			return false
		}
		if b[i] == nil {
			return true
		}
		if *a[i] < *b[i] {
			return false
		}
		if *a[i] > *b[i] {
			return true
		}
	}
	return false
}

type Album struct {
	ID           string
	CoverArtID   string
	Name         string
	Duration     time.Duration
	ArtistIDs    []string
	ArtistNames  []string
	Date         ItemDate
	ReissueDate  ItemDate
	Genres       []string
	TrackCount   int
	Favorite     bool
	ReleaseTypes ReleaseTypes
}

func (a *Album) YearOrZero() int {
	if a.Date.Year != nil {
		return *a.Date.Year
	}
	return 0
}

type AlbumWithTracks struct {
	Album
	Tracks []*Track
}

type AlbumInfo struct {
	Notes         string
	LastFmUrl     string
	MusicBrainzID string
}

type Artist struct {
	ID         string
	CoverArtID string
	Name       string
	Favorite   bool
	AlbumCount int
}

type ArtistWithAlbums struct {
	Artist
	Albums []*Album
}

type ArtistInfo struct {
	Biography      string
	LastFMUrl      string
	ImageURL       string
	SimilarArtists []*Artist
}

type Genre struct {
	Name       string
	AlbumCount int
	TrackCount int
}

type Track struct {
	ID               string
	CoverArtID       string
	ParentID         string
	Title            string
	Duration         time.Duration
	TrackNumber      int
	DiscNumber       int
	Genres           []string
	ArtistIDs        []string
	ArtistNames      []string
	AlbumArtistIDs   []string
	AlbumArtistNames []string
	ComposerIDs      []string
	ComposerNames    []string
	Album            string
	AlbumID          string
	Year             int
	Rating           int
	Favorite         bool
	Size             int64
	PlayCount        int
	LastPlayed       time.Time
	FilePath         string
	BitRate          int
	ContentType      string
	Comment          string
	BPM              int
	ReplayGain       ReplayGainInfo
	SampleRate       int
	BitDepth         int
	Extension        string
	Channels         int
	DateAdded        time.Time
}

type ReplayGainInfo struct {
	TrackGain float64
	AlbumGain float64
	TrackPeak float64
	AlbumPeak float64
}

type Playlist struct {
	ID          string
	CoverArtID  string
	Name        string
	Description string
	Public      bool
	Owner       string
	Duration    time.Duration
	TrackCount  int
}

type PlaylistWithTracks struct {
	Playlist
	Tracks []*Track
}

type Lyrics struct {
	Title  string      `json:"title"`
	Artist string      `json:"artist"`
	Synced bool        `json:"synced"`
	Lines  []LyricLine `json:"lines"`
}

type LyricLine struct {
	Text  string
	Start float64 // seconds
}

type SavedPlayQueue struct {
	Tracks   []*Track
	TrackPos int
	TimePos  int // seconds
}

type RadioStation struct {
	Name        string
	ID          string
	HomePageURL string
	StreamURL   string
}

type MediaItemType int

const (
	MediaItemTypeTrack MediaItemType = iota
	MediaItemTypeRadioStation
)

type MediaItemMetadata struct {
	Type       MediaItemType
	MIMEType   string
	ID         string
	Name       string
	Artists    []string
	ArtistIDs  []string
	Album      string
	AlbumID    string
	CoverArtID string
	Duration   time.Duration
}

type MediaItem interface {
	Metadata() MediaItemMetadata
	Copy() MediaItem
}

func (t *Track) Metadata() MediaItemMetadata {
	if t == nil {
		return MediaItemMetadata{}
	}
	return MediaItemMetadata{
		Type:       MediaItemTypeTrack,
		MIMEType:   t.ContentType,
		ID:         t.ID,
		Name:       t.Title,
		Artists:    t.ArtistNames,
		ArtistIDs:  t.ArtistIDs,
		Album:      t.Album,
		AlbumID:    t.AlbumID,
		CoverArtID: t.CoverArtID,
		Duration:   t.Duration,
	}
}

func (t *Track) Copy() MediaItem {
	if t == nil {
		return nil
	}
	new := *t
	return &new
}

func (r *RadioStation) Metadata() MediaItemMetadata {
	if r == nil {
		return MediaItemMetadata{}
	}
	return MediaItemMetadata{
		Type: MediaItemTypeRadioStation,
		ID:   r.ID,
		Name: r.Name,
	}
}

func (r *RadioStation) Copy() MediaItem {
	return r // no need to copy since RadioStations are immutable
}

type ContentType int

const (
	ContentTypeAlbum ContentType = iota
	ContentTypeArtist
	ContentTypePlaylist
	ContentTypeTrack
	ContentTypeGenre
	ContentTypeRadioStation
)

func (c ContentType) String() string {
	switch c {
	case ContentTypeAlbum:
		return "Album"
	case ContentTypeArtist:
		return "Artist"
	case ContentTypeTrack:
		return "Track"
	case ContentTypePlaylist:
		return "Playlist"
	case ContentTypeGenre:
		return "Genre"
	case ContentTypeRadioStation:
		return "Radio station"
	default:
		return "Unknown"
	}
}

type SearchResult struct {
	Name    string
	ID      string
	CoverID string
	Type    ContentType

	// for Album / Playlist: track count
	//     Artist / Genre: album count
	//     Track: length (seconds)
	// for Radio: none
	Size int

	// Unset for ContentTypes Artist, Playlist, Genre, and RadioStation
	ArtistName string

	// The actual item corresponding to this search result
	// *mediaprovider.Artist for ContentTypeArtist, etc
	Item any
}
