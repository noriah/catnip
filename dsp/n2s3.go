package dsp

import (
	"math"
	"time"

	"github.com/noriah/tavis/util"
)

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	prev   []float64
	time   time.Time
	window *util.MovingWindow
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(max int) *N2S3State {

	return &N2S3State{
		prev:   make([]float64, max),
		window: util.NewMovingWindow(20),
	}
}

// N2S3 does nora's not so special smoothing
func N2S3(bins []float64, count int, tick time.Time, state *N2S3State) {

	if state.time.IsZero() {
		state.time = time.Now().Add(-time.Second / 60)
	}

	var avg, _ = state.window.Update(tick.Sub(state.time).Seconds())

	var grav = 0.52 * math.Pow(60.0*avg, 1.75)

	for xBin := 0; xBin < count; xBin++ {
		bins[xBin] += state.prev[xBin] * grav
		state.prev[xBin] = bins[xBin]
	}

	state.time = tick
}
