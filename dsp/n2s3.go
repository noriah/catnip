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
	slowMax := int((5*hz)/float64(samples)) * 2
	fastMax := int((1*hz)/float64(samples)) * 2

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

		state.prevBins[xBin] = math.Max(0,
			n2s3Delta(bins[xBin], state.prevBins[xBin]))
		bins[xBin] = state.prevBins[xBin]

		if peak < bins[xBin] {
			peak = bins[xBin]
		}

		xBin++
	}

	if peak <= 0 {
		return
	}

	height--

	state.fastWindow.Update(peak)
	var vMean, vSD = state.slowWindow.Update(peak)

	if length := state.slowWindow.Len(); length >= state.fastWindow.Cap() {

		if math.Abs(state.fastWindow.Mean()-vMean) > (0.9 * vSD) {
			vMean, vSD = state.slowWindow.Drop(int(float64(length) * 0.75))
		}
	}

	// value to scale by to make conditions easier to base on
	var scale = height / math.Max(vMean+(1.5*vSD), math.SmallestNonzeroFloat64)

	xBin = 0
	for xBin < count {
		bins[xBin] = math.Min(height, bins[xBin]*scale)
		xBin++
	}
}

//
// SUBJECT TO CHANGE!!!
//
// n2s3Delta provided with a real and previous value will return the
// delta to add to the previous value.
func n2s3Delta(real, prev float64) float64 {

	var d = real - prev

	// if the real target is below our current value
	if d < 0.0 {

		if d > -0.5 {
			return prev - (d * d)
		}

		return prev + (d * 0.62)
	}

	if d < 0.5 {
		return prev + (d * d)
	}

	return prev + (d * 0.7)
}
