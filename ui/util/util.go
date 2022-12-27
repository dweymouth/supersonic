package util

import (
	"fmt"
	"math"
)

func SecondsToTimeString(s float64) string {
	if s < 0 {
		s = 0
	}
	sec := int(math.Round(s))
	min := sec / 60
	sec -= min * 60

	return fmt.Sprintf("%2d:%02d", min, sec)
}
