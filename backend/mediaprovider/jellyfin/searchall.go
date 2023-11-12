package jellyfin

import (
	"errors"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

func (s *jellyfinMediaProvider) SearchAll(searchQuery string, maxResults int) ([]*mediaprovider.SearchResult, error) {
	return nil, errors.New("unimplemented")
}
