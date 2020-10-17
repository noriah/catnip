package dsp

import (
	"math"
	"time"

	"github.com/noriah/tavis/util"
)

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	prev       []float64
	prevLevel  float64
	prevTime   time.Time
	timeWindow *util.MovingWindow
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(max int) *N2S3State {

	var state = &N2S3State{
		prev:       make([]float64, max),
		timeWindow: util.NewMovingWindow(60),
		prevLevel:  1,
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(bins []float64, num int, tick time.Time, state *N2S3State, factor float64) {

	if state.prevTime.IsZero() {
		state.prevTime = time.Now().Add(-time.Second / 60)
	}

	var avgTick, _ = state.timeWindow.Update(tick.Sub(state.prevTime).Seconds())
	if avgTick <= 0.0 || math.IsNaN(avgTick) {
		avgTick = (1 / 60)
	}

	var grav = factor * math.Pow(60*avgTick, 2.5)

	var dip = 1.0

	for xBin := 0; xBin < num; xBin++ {
		if bins[xBin] == 0.0 {
			continue
		}

		bins[xBin] += state.prev[xBin] * grav

		if bins[xBin] > dip {
			dip = bins[xBin]
		}

		state.prev[xBin] = bins[xBin] * (1 - ((1 / (bins[xBin] + 1)) / state.prevLevel))
	}

	state.prevLevel = dip

	state.prevTime = tick
}
