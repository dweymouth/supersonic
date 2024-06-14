package util

import "testing"

func TestSecondsToTimeString(t *testing.T) {
	inputs := []float64{
		57.2,
		3360,
		4800,
		4812,
		86401,
	}
	outputs := []string{
		"0:57",
		"56:00",
		"1 hr 20 min",
		"1 hr 20 min 12 sec",
		"1 day 1 sec",
	}
	for i, input := range inputs {
		if s := SecondsToTimeString(input); s != outputs[i] {
			t.Errorf("got %s, want %s", s, outputs[i])
		}
	}
}
