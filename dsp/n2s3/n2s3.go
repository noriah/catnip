// Package n2s3 contains the noriah's not so special smoother code
package n2s3

import (
	"math"
	"time"

	"github.com/noriah/tavis/util"
)

// State is the stateholder for N2S3
type State struct {
	sampleHz   float64
	sampleSize int

	prevBins []*attr

	prevTime time.Time
	// durWindow *util.MovingWindow

	scaleWindow *util.MovingWindow
}

// NewState returns a new N2S3 state.
func NewState(hz float64, samples int, max int) *State {
	var state = &State{
		sampleHz:   hz,
		sampleSize: samples,

		prevBins: make([]*attr, max),

		// durWindow:   util.NewMovingWindow(int(hz/float64(samples)) / 2),
		scaleWindow: util.NewMovingWindow(int(hz / float64(samples))),
	}
	for xBin := range state.prevBins {
		state.prevBins[xBin] = &attr{}
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(buffer []float64, count int, now time.Time, state *State) {

	// if state.prevTime.IsZero() {
	// 	state.prevTime = now.Add(
	// 		-time.Second / time.Duration(
	// 			int(state.sampleHz)/state.sampleSize))
	// }

	// state.durWindow.Update(now.Sub(state.prevTime).Seconds())
	// state.prevTime = now

	var peak = 0.0
	for xBin := 0; xBin < count; xBin++ {
		if peak < buffer[xBin] {
			peak = buffer[xBin]
		}
	}

	// Update our peak level. We want to scale everything to max = 1
	var scaleAvg, scaleSd = state.scaleWindow.Update(peak)

	// value to scale by to make conditions easier to base on
	var scale = math.Max(scaleAvg+(2*scaleSd), 1)

	// n2s3SecondPass(buffer, count, scale, state)

	for xBin := 0; xBin < count; xBin++ {
		// unscale our value  back to the original range
		buffer[xBin] = state.prevBins[xBin].addClamp(math.SmallestNonzeroFloat64, 1,
			n2s3Delta(buffer[xBin]/scale, state.prevBins[xBin].value))
	}

}

func n2s3Delta(r, p float64) float64 {
	var d = r - p
	var ad = math.Abs(d)

	// if we are at 0 height right now, fix that
	if p == 0 {
		return d
	}

	var max, min = r, p

	// If the real target is below our current value
	if min > max {
		max, min = min, max
		if ad >= 0.5 {
			return d * (ad / max)
		}
		return d * 0.5
	}

	if ad >= 0.2 {
		return d * 0.95
	}

	return d * 0.5
}
