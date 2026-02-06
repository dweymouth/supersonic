package backend

import "math"

// AutoEQ uses 10 bands at these frequencies (in Hz)
var autoEQFreqs = []float64{31.25, 62.5, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}

// Supersonic uses 15 bands at these frequencies (in Hz)
var supersonicFreqs = []float64{25, 40, 63, 100, 160, 250, 400, 630, 1000, 1600, 2500, 4000, 6300, 10000, 16000}

// InterpolateAutoEQTo15Band converts a 10-band AutoEQ profile to Supersonic's 15-band ISO equalizer.
// Uses logarithmic frequency positioning with linear dB interpolation.
//
// Parameters:
//   - autoEQGains: Array of 10 gain values (dB) from AutoEQ at 31, 62, 125, 250, 500, 1k, 2k, 4k, 8k, 16k Hz
//
// Returns:
//   - Array of 15 gain values (dB) for Supersonic at 25, 40, 63, 100, 160, 250, 400, 630, 1k, 1.6k, 2.5k, 4k, 6.3k, 10k, 16k Hz
func InterpolateAutoEQTo15Band(autoEQGains [10]float64) [15]float64 {
	var result [15]float64

	for i, targetFreq := range supersonicFreqs {
		// Find the surrounding AutoEQ bands for this target frequency
		lowerIdx, upperIdx := findSurroundingBands(targetFreq)

		if lowerIdx == upperIdx {
			// Exact match - use the AutoEQ gain directly
			result[i] = autoEQGains[lowerIdx]
		} else if lowerIdx == -1 {
			// Target frequency is below the lowest AutoEQ band (25 Hz < 31.25 Hz)
			// Extrapolate using the first two AutoEQ bands
			result[i] = extrapolateBelow(targetFreq, autoEQGains)
		} else if upperIdx == -1 {
			// Target frequency is above the highest AutoEQ band (should not happen with our ranges)
			// Use the highest AutoEQ gain
			result[i] = autoEQGains[len(autoEQGains)-1]
		} else {
			// Interpolate between two AutoEQ bands
			fLow := autoEQFreqs[lowerIdx]
			fHigh := autoEQFreqs[upperIdx]
			gLow := autoEQGains[lowerIdx]
			gHigh := autoEQGains[upperIdx]

			// Calculate logarithmic position between the two bands
			// t = log(targetFreq/fLow) / log(fHigh/fLow)
			t := math.Log(targetFreq/fLow) / math.Log(fHigh/fLow)

			// Linear interpolation of gain values
			result[i] = gLow + t*(gHigh-gLow)
		}
	}

	return result
}

// findSurroundingBands finds the AutoEQ band indices that surround the target frequency.
// Returns (idx, idx) if there's an exact match, (lowerIdx, upperIdx) if between bands,
// (-1, 0) if below all bands, or (lastIdx, -1) if above all bands.
func findSurroundingBands(targetFreq float64) (lowerIdx, upperIdx int) {
	const epsilon = 0.01 // Tolerance for floating point comparison

	// Check if below all bands
	if targetFreq < autoEQFreqs[0]-epsilon {
		return -1, 0
	}

	// Check if above all bands
	if targetFreq > autoEQFreqs[len(autoEQFreqs)-1]+epsilon {
		return len(autoEQFreqs) - 1, -1
	}

	// Find surrounding bands
	for i := 0; i < len(autoEQFreqs)-1; i++ {
		// Check for exact match
		if math.Abs(targetFreq-autoEQFreqs[i]) < epsilon {
			return i, i
		}

		// Check if between this band and the next
		if targetFreq > autoEQFreqs[i] && targetFreq < autoEQFreqs[i+1] {
			return i, i + 1
		}
	}

	// Check for exact match with last band
	if math.Abs(targetFreq-autoEQFreqs[len(autoEQFreqs)-1]) < epsilon {
		return len(autoEQFreqs) - 1, len(autoEQFreqs) - 1
	}

	// Should not reach here with valid input
	return len(autoEQFreqs) - 1, -1
}

// extrapolateBelow extrapolates the gain for frequencies below the lowest AutoEQ band.
// Uses the slope between the first two AutoEQ bands.
func extrapolateBelow(targetFreq float64, autoEQGains [10]float64) float64 {
	// Use the slope between the first two bands to extrapolate
	f1 := autoEQFreqs[0]
	f2 := autoEQFreqs[1]
	g1 := autoEQGains[0]
	g2 := autoEQGains[1]

	// Calculate the slope in log-frequency space
	slope := (g2 - g1) / math.Log(f2/f1)

	// Extrapolate
	return g1 + slope*math.Log(targetFreq/f1)
}

// Interpolate15BandTo10Band converts a 15-band ISO equalizer to 10-band format.
// Uses logarithmic frequency positioning with linear dB interpolation.
func Interpolate15BandTo10Band(gains15Band [15]float64) [10]float64 {
	var result [10]float64

	for i, targetFreq := range autoEQFreqs {
		// Find the surrounding 15-band frequencies for this target
		lowerIdx, upperIdx := findSurrounding15Bands(targetFreq)

		if lowerIdx == upperIdx {
			// Exact match
			result[i] = gains15Band[lowerIdx]
		} else if lowerIdx == -1 {
			// Below lowest band - use first band value
			result[i] = gains15Band[0]
		} else if upperIdx == -1 {
			// Above highest band - use last band value
			result[i] = gains15Band[len(gains15Band)-1]
		} else {
			// Interpolate between two 15-band frequencies
			fLow := supersonicFreqs[lowerIdx]
			fHigh := supersonicFreqs[upperIdx]
			gLow := gains15Band[lowerIdx]
			gHigh := gains15Band[upperIdx]

			// Logarithmic position
			t := math.Log(targetFreq/fLow) / math.Log(fHigh/fLow)

			// Linear interpolation of gain
			result[i] = gLow + t*(gHigh-gLow)
		}
	}

	return result
}

// findSurrounding15Bands finds the 15-band indices that surround the target frequency
func findSurrounding15Bands(targetFreq float64) (lowerIdx, upperIdx int) {
	const epsilon = 0.01

	if targetFreq < supersonicFreqs[0]-epsilon {
		return -1, 0
	}

	if targetFreq > supersonicFreqs[len(supersonicFreqs)-1]+epsilon {
		return len(supersonicFreqs) - 1, -1
	}

	for i := 0; i < len(supersonicFreqs)-1; i++ {
		if math.Abs(targetFreq-supersonicFreqs[i]) < epsilon {
			return i, i
		}

		if targetFreq > supersonicFreqs[i] && targetFreq < supersonicFreqs[i+1] {
			return i, i + 1
		}
	}

	if math.Abs(targetFreq-supersonicFreqs[len(supersonicFreqs)-1]) < epsilon {
		return len(supersonicFreqs) - 1, len(supersonicFreqs) - 1
	}

	return len(supersonicFreqs) - 1, -1
}
