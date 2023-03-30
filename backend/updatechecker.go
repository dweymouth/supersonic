package backend

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type UpdateChecker struct {
	OnUpdatedVersionFound func(releaseURL string)

	foundUpdate      bool
	latestReleaseURL string
	appVersionTag    string
	lastCheckedTag   *string
}

func NewUpdateChecker(appVersionTag, latestReleaseURL string, lastCheckedTag *string) UpdateChecker {
	return UpdateChecker{
		appVersionTag:    appVersionTag,
		latestReleaseURL: latestReleaseURL,
		lastCheckedTag:   lastCheckedTag,
	}
}

func (u *UpdateChecker) Start(ctx context.Context, interval time.Duration) {
	go func() {
		u.checkForUpdate() // check once at startup
		t := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				u.checkForUpdate()
			}
		}
	}()
}

func (u *UpdateChecker) UpdateAvailable() bool {
	return u.foundUpdate
}

func (u *UpdateChecker) checkForUpdate() {
	t := u.latestVersionTag()
	if t != "" && t != *u.lastCheckedTag {
		u.foundUpdate = true
		*u.lastCheckedTag = t
		if u.OnUpdatedVersionFound != nil {
			u.OnUpdatedVersionFound(u.latestReleaseURL)
		}
	}
}

func (u *UpdateChecker) latestVersionTag() string {
	resp, err := http.Head(u.latestReleaseURL)
	if err != nil {
		log.Printf("failed to check for newest version: %s", err.Error())
		return ""
	}
	url := resp.Request.URL.String()
	url = strings.TrimSuffix(url, "/")
	idx := strings.LastIndex(url, "/")
	if idx >= len(url)-1 {
		return ""
	}
	return url[idx+1:]
}
