package dsp

import (
	"math"

	"github.com/noriah/tavis/util"
)

// Scaling Constants
const (

	// ScalingFastWindow in seconds
	ScalingSlowWindow = 10

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	ScalingResetDeviation = 1
)

type ScaleState struct {
	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow
}

func NewScaleState(hz float64, samples int) *ScaleState {

	slowMax := int((ScalingSlowWindow*hz)/float64(samples)) * 2
	fastMax := int((ScalingFastWindow*hz)/float64(samples)) * 2

	return &ScaleState{
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
	}
}

// Scale scales the data
func Scale(bins []float64, count int, height float64, state *ScaleState) {

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

		if math.Abs(state.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
			state.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))

			vMean, vSD = state.slowWindow.Stats()
		}
	}

	var vMag = math.Max(vMean+(2*vSD), 1)

	height--
	for xBin := 0; xBin < count; xBin++ {
		bins[xBin] = math.Min(height, (bins[xBin]/vMag)*height)
	}
}
