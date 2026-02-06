package backend

import (
	"math"
	"testing"
)

func TestInterpolateAutoEQTo15Band_FlatProfile(t *testing.T) {
	// Test with a flat profile (all zeros)
	flatProfile := [10]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	result := InterpolateAutoEQTo15Band(flatProfile)

	for i, gain := range result {
		if math.Abs(gain) > 0.001 {
			t.Errorf("Flat profile should result in zero gains, got %f at index %d", gain, i)
		}
	}
}

func TestInterpolateAutoEQTo15Band_ExactMatches(t *testing.T) {
	// Test that exact frequency matches use the AutoEQ gain directly
	// AutoEQ bands: 31, 62, 125, 250, 500, 1k, 2k, 4k, 8k, 16k Hz
	// Supersonic bands: 25, 40, 63, 100, 160, 250, 400, 630, 1k, 1.6k, 2.5k, 4k, 6.3k, 10k, 16k Hz
	// Exact matches: 250 (idx 5→3), 1k (idx 5→8), 4k (idx 7→11), 16k (idx 9→14)

	testProfile := [10]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := InterpolateAutoEQTo15Band(testProfile)

	tests := []struct {
		supersonicIdx int
		autoEQIdx     int
		expectedGain  float64
	}{
		{5, 3, 4},   // 250 Hz
		{8, 5, 6},   // 1000 Hz
		{11, 7, 8},  // 4000 Hz
		{14, 9, 10}, // 16000 Hz
	}

	for _, tt := range tests {
		if math.Abs(result[tt.supersonicIdx]-tt.expectedGain) > 0.001 {
			t.Errorf("Exact match at index %d (AutoEQ idx %d): expected %f, got %f",
				tt.supersonicIdx, tt.autoEQIdx, tt.expectedGain, result[tt.supersonicIdx])
		}
	}
}

func TestInterpolateAutoEQTo15Band_Interpolation(t *testing.T) {
	// Test interpolation between bands
	// Use a profile with known values to verify interpolation
	testProfile := [10]float64{0, 0, 0, 0, 0, 10, 0, 0, 0, 0}
	// Only 1000 Hz (index 5) has gain of 10 dB

	result := InterpolateAutoEQTo15Band(testProfile)

	// Check that 1000 Hz (index 8) has the exact value
	if math.Abs(result[8]-10) > 0.001 {
		t.Errorf("1000 Hz should be 10 dB, got %f", result[8])
	}

	// Check that nearby frequencies are interpolated (should be between 0 and 10)
	// 630 Hz (index 7) should be between 500 Hz (0 dB) and 1000 Hz (10 dB)
	if result[7] < 0 || result[7] > 10 {
		t.Errorf("630 Hz interpolation out of range: %f", result[7])
	}

	// 1600 Hz (index 9) should be between 1000 Hz (10 dB) and 2000 Hz (0 dB)
	if result[9] < 0 || result[9] > 10 {
		t.Errorf("1600 Hz interpolation out of range: %f", result[9])
	}
}

func TestInterpolateAutoEQTo15Band_Extrapolation(t *testing.T) {
	// Test extrapolation for 25 Hz (below lowest AutoEQ band of 31.25 Hz)
	testProfile := [10]float64{5, 10, 0, 0, 0, 0, 0, 0, 0, 0}
	// 31.25 Hz = 5 dB, 62.5 Hz = 10 dB

	result := InterpolateAutoEQTo15Band(testProfile)

	// 25 Hz should be extrapolated below 31.25 Hz
	// Since the slope is positive (5 to 10), 25 Hz should be < 5 dB
	if result[0] > 5 {
		t.Errorf("25 Hz extrapolation should be < 5 dB, got %f", result[0])
	}

	// It should be a reasonable value (not too extreme)
	if math.Abs(result[0]) > 20 {
		t.Errorf("25 Hz extrapolation too extreme: %f", result[0])
	}
}

func TestInterpolateAutoEQTo15Band_RealProfile(t *testing.T) {
	// Test with a realistic profile shape (bass and treble boost)
	// Approximating a V-shaped response
	testProfile := [10]float64{6, 5, 3, 1, 0, 0, 1, 3, 5, 6}

	result := InterpolateAutoEQTo15Band(testProfile)

	// Verify output is reasonable
	if len(result) != 15 {
		t.Errorf("Expected 15 bands, got %d", len(result))
	}

	// Check that interpolated values are between neighboring AutoEQ values
	// For example, 40 Hz (index 1) should be between 31.25 Hz and 62.5 Hz values
	if result[1] < math.Min(testProfile[0], testProfile[1])-1 ||
		result[1] > math.Max(testProfile[0], testProfile[1])+1 {
		t.Errorf("40 Hz interpolation out of reasonable range: %f (between %f and %f)",
			result[1], testProfile[0], testProfile[1])
	}
}

func TestInterpolateAutoEQTo15Band_MonotonicPreservation(t *testing.T) {
	// Test that monotonic sections are preserved
	// If AutoEQ gains are monotonically increasing, interpolated values should also increase
	monotonicProfile := [10]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	result := InterpolateAutoEQTo15Band(monotonicProfile)

	// Check general increasing trend (allowing for some extrapolation variance at edges)
	for i := 2; i < len(result)-1; i++ {
		if result[i] < result[i-1]-0.5 {
			t.Errorf("Monotonicity not preserved at index %d: %f < %f",
				i, result[i], result[i-1])
		}
	}
}

func TestFindSurroundingBands_ExactMatches(t *testing.T) {
	tests := []struct {
		freq          float64
		expectedLower int
		expectedUpper int
	}{
		{31.25, 0, 0},
		{62.5, 1, 1},
		{125, 2, 2},
		{250, 3, 3},
		{1000, 5, 5},
		{4000, 7, 7},
		{16000, 9, 9},
	}

	for _, tt := range tests {
		lower, upper := findSurroundingBands(tt.freq)
		if lower != tt.expectedLower || upper != tt.expectedUpper {
			t.Errorf("findSurroundingBands(%f): expected (%d, %d), got (%d, %d)",
				tt.freq, tt.expectedLower, tt.expectedUpper, lower, upper)
		}
	}
}

func TestFindSurroundingBands_BetweenBands(t *testing.T) {
	tests := []struct {
		freq          float64
		expectedLower int
		expectedUpper int
	}{
		{40, 0, 1},    // Between 31.25 and 62.5
		{100, 1, 2},   // Between 62.5 and 125
		{500, 4, 4},   // Exact match at 500
		{2000, 6, 6},  // Exact match at 2000
		{10000, 8, 9}, // Between 8000 and 16000
	}

	for _, tt := range tests {
		lower, upper := findSurroundingBands(tt.freq)
		if lower != tt.expectedLower || upper != tt.expectedUpper {
			t.Errorf("findSurroundingBands(%f): expected (%d, %d), got (%d, %d)",
				tt.freq, tt.expectedLower, tt.expectedUpper, lower, upper)
		}
	}
}

func TestFindSurroundingBands_BelowRange(t *testing.T) {
	lower, upper := findSurroundingBands(25)
	if lower != -1 || upper != 0 {
		t.Errorf("findSurroundingBands(25): expected (-1, 0), got (%d, %d)", lower, upper)
	}
}
