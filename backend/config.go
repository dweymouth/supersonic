package backend

import (
	"os"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml"
)

type ServerConfig struct {
	ID         uuid.UUID
	Nickname   string
	Hostname   string
	Username   string
	LegacyAuth bool
	Default    bool
}

type AppConfig struct {
	WindowWidth  int
	WindowHeight int
}

type AlbumPageConfig struct {
	TracklistColumns []string
}

type AlbumsPageConfig struct {
	SortOrder string
}

type ArtistPageConfig struct {
	InitialView      string
	TracklistColumns []string
}

type FavoritesPageConfig struct {
	InitialView      string
	TracklistColumns []string
}

type NowPlayingPageConfig struct {
	TracklistColumns []string
}

type PlaylistPageConfig struct {
	TracklistColumns []string
}

type LocalPlaybackConfig struct {
	AudioExclusive      bool
	InMemoryCacheSizeMB int
	Volume              int
}

type ScrobbleConfig struct {
	Enabled              bool
	ThresholdTimeSeconds int
	ThresholdPercent     int
}

type ReplayGainConfig struct {
	Mode            string
	PreampGainDB    float64
	PreventClipping bool
}

type Config struct {
	Application    AppConfig
	Servers        []*ServerConfig
	AlbumPage      AlbumPageConfig
	AlbumsPage     AlbumsPageConfig
	ArtistPage     ArtistPageConfig
	FavoritesPage  FavoritesPageConfig
	NowPlayingPage NowPlayingPageConfig
	PlaylistPage   PlaylistPageConfig
	LocalPlayback  LocalPlaybackConfig
	Scrobbling     ScrobbleConfig
	ReplayGain     ReplayGainConfig
}

func DefaultConfig() *Config {
	return &Config{
		Application: AppConfig{
			WindowWidth:  1000,
			WindowHeight: 800,
		},
		AlbumPage: AlbumPageConfig{
			TracklistColumns: []string{"Artist", "Time", "Plays", "Favorite"},
		},
		AlbumsPage: AlbumsPageConfig{
			SortOrder: string(AlbumSortRecentlyAdded),
		},
		ArtistPage: ArtistPageConfig{
			InitialView:      "Discography",
			TracklistColumns: []string{"Album", "Time", "Plays", "Favorite"},
		},
		FavoritesPage: FavoritesPageConfig{
			TracklistColumns: []string{"Artist", "Album", "Time", "Plays"},
			InitialView:      "Albums",
		},
		NowPlayingPage: NowPlayingPageConfig{
			TracklistColumns: []string{"Artist", "Album", "Time", "Plays"},
		},
		PlaylistPage: PlaylistPageConfig{
			TracklistColumns: []string{"Artist", "Album", "Time", "Plays"},
		},
		LocalPlayback: LocalPlaybackConfig{
			AudioExclusive:      false,
			InMemoryCacheSizeMB: 30,
			Volume:              100,
		},
		Scrobbling: ScrobbleConfig{
			Enabled:              true,
			ThresholdTimeSeconds: 240,
			ThresholdPercent:     50,
		},
		ReplayGain: ReplayGainConfig{
			Mode:            ReplayGainNone,
			PreampGainDB:    0.0,
			PreventClipping: true,
		},
	}
}

func ReadConfigFile(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := DefaultConfig()
	if err := toml.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) GetDefaultServer() *ServerConfig {
	for _, s := range c.Servers {
		if s.Default {
			return s
		}
	}
	if len(c.Servers) > 0 {
		return c.Servers[0]
	}
	return nil
}

func (c *Config) SetDefaultServer(serverID uuid.UUID) {
	var found bool
	for _, s := range c.Servers {
		f := s.ID == serverID
		if f {
			found = true
		}
		s.Default = f
	}
	if !found && len(c.Servers) > 0 {
		c.Servers[0].Default = true
	}
}

func (c *Config) AddServer(nickname, hostname, username string, legacyAuth bool) *ServerConfig {
	s := &ServerConfig{
		ID:         uuid.New(),
		Nickname:   nickname,
		Hostname:   hostname,
		Username:   username,
		LegacyAuth: legacyAuth,
	}
	c.Servers = append(c.Servers, s)
	return s
}

func (c *Config) WriteConfigFile(filepath string) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	os.WriteFile(filepath, b, 0644)

	return nil
}
