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
	slowMax := int((6*hz)/float64(samples)) * 2
	fastMax := int((2*hz)/float64(samples)) * 2

	return &N2S3State{
		prevBins:   make([]float64, max),
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
	}
}

// N2S3 does nora's not so special smoothing
func N2S3(bins []float64, count int, height float64, state *N2S3State) {

	var peak = 0.0

	for xBin := 0; xBin < count; xBin++ {

		state.prevBins[xBin] = math.Max(0,
			n2s3Next(bins[xBin], state.prevBins[xBin]))

		bins[xBin] = state.prevBins[xBin]

		if peak < bins[xBin] {
			peak = bins[xBin]
		}
	}

	if peak <= 0 {
		return
	}

	height--

	state.fastWindow.Update(peak)
	var vMean, vSD = state.slowWindow.Update(peak)

	if length := state.slowWindow.Len(); length >= state.fastWindow.Cap() {

		if math.Abs(state.fastWindow.Mean()-vMean) > (0.9 * vSD) {
			vMean, vSD = state.slowWindow.Drop(int(float64(length) * 0.65))
		}
	}

	// value to scale by to make conditions easier to base on
	var scale = height / math.Max(vMean+(1.5*vSD), math.SmallestNonzeroFloat64)

	for xBin := 0; xBin < count; xBin++ {
		bins[xBin] = math.Min(height, bins[xBin]*scale)
	}
}

//
// SUBJECT TO CHANGE!!!
//
// n2s3Next provided with a real and previous value will return the
// delta to add to the previous value.
func n2s3Next(real, prev float64) float64 {
	if real == 0 {

		return prev * 0.5
	}

	var d = real - prev

	if d > 0.0 {
		return prev + math.Min(d*0.999, math.Max((d/real)*1.5, d*0.5))
	}

	return prev + math.Max(d*0.65, prev/d)
}
