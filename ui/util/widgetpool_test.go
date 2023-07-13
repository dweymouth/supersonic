package util

import (
	"testing"
	"time"
)

func Test_WidgetPool_CleanupExpiredItems(t *testing.T) {
	now := time.Now()
	threeMinAgo := now.Add(-3 * time.Minute)
	w := &WidgetPool{
		pools: [][]pooledWidget{
			{
				{releasedAt: threeMinAgo.UnixMilli()},
				{releasedAt: threeMinAgo.UnixMilli()},
				{releasedAt: threeMinAgo.UnixMilli()},
			},
		},
	}
	w.cleanUpExpiredItems()
	if l := len(w.pools[0]); l != 1 {
		t.Errorf("Expected one widget in pool after cleanup, got %d", l)
	}
}
