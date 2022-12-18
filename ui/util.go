package ui

import "fmt"

func SecondsToTimeString(s float64) string {
	if s < 0 {
		s = 0
	}
	min := int(s / 60)
	sec := int(s - float64(min*60))

	return fmt.Sprintf("%2d:%02d", min, sec)
}
