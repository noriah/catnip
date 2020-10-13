package dsp

import "math"

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	prev []float64
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(max int) *N2S3State {

	return &N2S3State{
		prev: make([]float64, max),
	}
}

// N2S3 does nora's not so special smoothing
func N2S3(bins []float64, count int, state *N2S3State) {
	for xBin := 0; xBin < count; xBin++ {
		// until we touch it, bins[xBin] is our real value
		// state.prev[xBin] is our previous value
		state.prev[xBin] = math.Max(0, n2s3Next(bins[xBin], state.prev[xBin]))
		bins[xBin] = state.prev[xBin]
	}
}

//
// SUBJECT TO CHANGE!!!
//
// n2s3Next, provided with a real and previous value will return the
// next value.
func n2s3Next(real, prev float64) float64 {
	// if our real value is 0, head towards 0
	if real == 0 {
		return prev * 0.5
	}

	// d is the delta between real and previous values
	var d = real - prev

	// if our previous value or our delta is 0, return our current real value
	if d == 0 || prev == 0 {
		return real
	}

	// if positive
	if d > 0.0 {
		// return the previous value plus the Minimum of:
		// 99.9% of delta | the Maximum of:
		// 150% of (delta / real) | 50 % of delta
		return prev + math.Min(d*0.999, math.Max((d/real)*1.5, d*0.5))
	}

	// if negative

	// return the previous value plus the Maximum of:
	// 65% of delta | previous / delta
	return prev + math.Max(d*0.65, prev/d)
}
