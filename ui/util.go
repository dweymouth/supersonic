package ui

import (
	"fmt"
	"math"
)

func SecondsToTimeString(s float64) string {
	if s < 0 {
		s = 0
	}
	min := int(s / 60)
	sec := int(math.Round(s - float64(min*60)))

	return fmt.Sprintf("%2d:%02d", min, sec)
}
