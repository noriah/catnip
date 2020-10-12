package dsp

import (
	"math"

	"github.com/noriah/tavis/util"
)

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	prevBins   []float64
	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(hz float64, samples int, max int) *N2S3State {
	slowMax := int((5 * hz) / float64(samples))
	fastMax := int((1 * hz) / float64(samples))

	return &N2S3State{
		prevBins:   make([]float64, max),
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
	}
}

// N2S3 does nora's not so special smoothing
func N2S3(bins []float64, count int, height float64, state *N2S3State) {

	var xBin = 0
	var peak = 0.0

	for xBin < count {
		if peak < bins[xBin] {
			peak = bins[xBin]
		}

		xBin++
	}

	if peak <= 0 {
		return
	}

	state.fastWindow.Update(peak)
	var vMean, vSD = state.slowWindow.Update(peak)

	if length := state.slowWindow.Len(); length >= state.fastWindow.Cap() {

		if math.Abs(state.fastWindow.Mean()-vMean) > (0.9 * vSD) {
			vMean, vSD = state.slowWindow.Drop(
				int(float64(length) * 0.7))
		}
	}

	// value to scale by to make conditions easier to base on
	var scale = math.Max(vMean+(2*vSD), 1)

	if scale < 1 {
		scale = 1
	}

	height--

	xBin = 0

	for xBin < count {
		var next = math.Max(
			0, math.Min(
				1, state.prevBins[xBin]+n2s3Delta(
					bins[xBin]/scale,
					state.prevBins[xBin])))

		bins[xBin] = math.Min(height, next*height)

		state.prevBins[xBin] = next

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
	if prev <= 0 {
		return real
	}

	var d = real - prev

	if d == 0 {
		return 0
	}

	// if the real target is below our current value
	if d < 0.0 {
		// filter large delta
		if d < -0.2 {
			// filter medium delta
			// if d < -0.8 {
			// large delta
			return d * 0.6
			// }
			// medium delta
			// return d * (d / -prev) * 1.2
		}
		// small delta
		return d * ((1 + d) / 2.2)
	}

	// filter small delta
	if d > 0.1 {

		if d > 0.8 {
			return d * 0.9
		}
		// filter medium delta
		// if d > 0.9 {
		// large delta
		// return d * 0.8
		// }
		// medium delta
		return d * 0.8
	}
	// small delta
	return d * ((1 - d) / 0.95)
}
