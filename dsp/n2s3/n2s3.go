// Package n2s3 contains the noriah's not so special smoother code
package n2s3

import (
	"math"
	"time"

	"github.com/noriah/tavis/util"
)

// consts
const (
	Smoothness = 1
)

// State is the stateholder for N2S3
type State struct {
	sampleHz   float64
	sampleSize int

	sum         attr
	avg         attr
	max         attr
	min         attr
	energy      attr
	sumDelta    attr
	avgDelta    attr
	maxDelta    attr
	minDelta    attr
	sumAbsDelta attr
	avgAbsDelta attr
	maxAbsDelta attr
	minAbsDelta attr

	prevBins []attr

	prevTime  time.Time
	durWindow *util.MovingWindow
}

// NewState returns a new N2S3 state.
func NewState(hz float64, samples int, max int) *State {
	return &State{
		sampleHz:   hz,
		sampleSize: samples,

		prevBins: make([]attr, max),

		durWindow: util.NewMovingWindow(int(hz/float64(samples)) / 2),
	}
}

// N2S3 does nora's not so special smoothing
func N2S3(dest []float64, count int, now time.Time, state *State) {

	if state.prevTime.IsZero() {
		state.prevTime = now.Add(
			-time.Second / time.Duration(
				int(state.sampleHz)/state.sampleSize))
	}

	// var tAvg, _ = state.durWindow.Update(now.Sub(state.prevTime).Seconds())

	n2s3FirstPass(dest, count, state)

	n2s3EnergyHelper(count, state)

	for xBin := 0; xBin < count; xBin++ {
		dest[xBin] = state.prevBins[xBin].addZero(
			n2s3DeltaFrac(state, dest[xBin], state.prevBins[xBin].value))
	}

	state.prevTime = time.Now()
}

func n2s3FirstPass(bins []float64, count int, state *State) {
	var fCount = float64(count)

	var (
		sum         float64
		max         float64
		min         float64 = math.MaxFloat64
		sumDelta    float64
		maxDelta    float64 = -math.MaxFloat64
		minDelta    float64 = math.MaxFloat64
		sumAbsDelta float64
		maxAbsDelta float64
		minAbsDelta float64 = math.MaxFloat64
	)

	for xBin := 0; xBin < count; xBin++ {
		sum += bins[xBin]

		if bins[xBin] > max {
			max = bins[xBin]
		}

		if bins[xBin] < min {
			min = bins[xBin]
		}

		var del = bins[xBin] - state.prevBins[xBin].value
		sumDelta += del

		if del > maxDelta {
			maxDelta = del
		}

		if del < minDelta {
			minDelta = del
		}

		var adel = math.Abs(del)
		sumAbsDelta += adel

		if adel > maxAbsDelta {
			maxAbsDelta = adel
		}

		if adel < minAbsDelta {
			minAbsDelta = adel
		}
	}

	state.sum.set(sum)
	state.avg.set(sum / fCount)
	state.max.set(max)
	state.min.set(min)
	state.sumDelta.set(sumDelta)
	state.avgDelta.set(sumDelta / fCount)
	state.maxDelta.set(maxDelta)
	state.minDelta.set(minDelta)
	state.sumAbsDelta.set(sumAbsDelta)
	state.avgAbsDelta.set(sumAbsDelta / fCount)
	state.maxAbsDelta.set(maxAbsDelta)
	state.minAbsDelta.set(minAbsDelta)
}

func n2s3EnergyHelper(count int, state *State) {
	var energy float64

	state.energy.set(energy)
}

func n2s3DeltaFrac(state *State, r, p float64) float64 {
	var d, ad, dPct, _, _, pct = n2s3DeltaHelper(state.avgAbsDelta, p)

	// use the ratio of change / peak height of transition
	// as percent value

	// var absEnergy = math.Abs(energy)

	// percentage of activity this bar counts for

	if ad < 0.001 {
		return d
	}

	if dPct <= 0.1 && ad <= 0.5 {
		return d * ad
	}

	if pct > 0.1 && pct < 1 {
		// return d * pct * math.Pow(2*pct, 0.5)
	}

	if r < p {

		if pct < 0.9 {
			pct *= 2
		}
		return d * math.Pow(5*pct, 0.01)
	}

	if ad > 0.1 {
		return d * math.Pow(pct, 1)
	}

	return d
}

func n2s3DeltaHelper(a, r, p float64) (del, adel, dPct, max, min, pct float64) {
	del = r - p
	adel = math.Abs(del)
	dPct = del / a
	max = math.Max(r, p)
	min = max - adel
	pct = adel / max

	return
}
