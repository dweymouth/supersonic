package player

import (
	"fmt"
	"strings"
)

type WidthType int

const (
	WidthTypeHz WidthType = iota
	WidthTypeKhz
	WidthTypeQ
	WidthTypeOctave
	WidthTypeSlope
)

type EqualizerBand struct {
	Frequency int
	Gain      float64
	Width     float64
	WidthType WidthType
}

type EqualizerCurve []EqualizerBand

func (e EqualizerCurve) String() string {
	var sb strings.Builder
	for i, band := range e {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(band.String())
	}
	return sb.String()
}

func (e EqualizerBand) String() string {
	return fmt.Sprintf("equalizer=f=%d:g=%0.2f:t=%s:w=%0.2f",
		e.Frequency, e.Gain, e.WidthType.String(), e.Width)
}

func (w WidthType) String() string {
	switch w {
	case WidthTypeHz:
		return "h"
	case WidthTypeKhz:
		return "k"
	case WidthTypeQ:
		return "q"
	case WidthTypeOctave:
		return "o"
	case WidthTypeSlope:
		return "s"
	}
	return "x" // not reached
}
