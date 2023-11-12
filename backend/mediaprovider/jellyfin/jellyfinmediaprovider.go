package jellyfin

import (
	"errors"

	"github.com/dweymouth/go-jellyfin"
)

type jellyfinMediaProvider struct {
	client          *jellyfin.Client
	prefetchCoverCB func(coverArtID string)
}

func (j *jellyfinMediaProvider) SetPrefetchCoverCallback(cb func(coverArtID string)) {
	j.prefetchCoverCB = cb
}

func (jellyfinMediaProvider) CreatePlaylist(name string, trackIDs []string) error {
	return errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) DeletePlaylist(id string) error {
	return j.client.DeletePlaylist(id)
}
