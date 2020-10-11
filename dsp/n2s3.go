package dsp

import (
	"math"
	"time"

	"github.com/noriah/tavis/util"
)

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	prevBins    []float64
	scaleWindow *util.MovingWindow
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(hz float64, samples int, max int) *N2S3State {
	var state = &N2S3State{
		prevBins:    make([]float64, max),
		scaleWindow: util.NewMovingWindow(int(hz/float64(samples)) / 2),
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(buffer []float64, count int, now time.Time, state *N2S3State) {

	var xBin = 0
	var peak = 0.0

	for xBin < count {
		if peak < buffer[xBin] {
			peak = buffer[xBin]
		}

		xBin++
	}

	if peak <= 0 {
		return
	}

	// Update our peak level. We want to scale everything to max = 1
	var scaleAvg, scaleSd = state.scaleWindow.Update(peak)

	// value to scale by to make conditions easier to base on
	var scale = math.Max(scaleAvg+(2*scaleSd), 1)

	if scale > peak*3 {
		state.scaleWindow.Drop(6)
	}

	xBin = 0
	for xBin < count {
		// unscale our value  back to the original range
		state.prevBins[xBin] = math.Max(
			0, math.Min(
				1, state.prevBins[xBin]+n2s3Delta(
					buffer[xBin]/scale,
					state.prevBins[xBin])))

		buffer[xBin] = scale * state.prevBins[xBin]
		xBin++
	}
}

//
// SUBJECT TO CHANGE!!!
//
// n2s3Delta provided with a real and previous value will return the
// delta to add to the previous value.
func n2s3Delta(real, prev float64) float64 {

	// if we are at 0 height right now, fix that
	if prev == 0 {
		return real
	}

	var d = real - prev

	// if the real target is below our current value
	if d < 0 {

		if d <= -0.01 {
			if d <= -0.7 {
				return d
			}
			return d * 0.5
		}

		return d * 0.2

	}

	if d >= 0.05 {

		if d >= 0.7 {
			return d
		}

		return d * 0.5
	}

	return d * d

}
