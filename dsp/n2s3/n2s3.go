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

	prevBins []*attr

	prevTime  time.Time
	durWindow *util.MovingWindow

	scaleWindow *util.MovingWindow
}

// NewState returns a new N2S3 state.
func NewState(hz float64, samples int, max int) *State {
	var state = &State{
		sampleHz:   hz,
		sampleSize: samples,

		prevBins: make([]*attr, max),

		durWindow:   util.NewMovingWindow(int(hz/float64(samples)) / 2),
		scaleWindow: util.NewMovingWindow(100),
	}
	for xBin := range state.prevBins {
		state.prevBins[xBin] = &attr{}
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(dest []float64, count int, now time.Time, state *State) {

	if state.prevTime.IsZero() {
		state.prevTime = now.Add(
			-time.Second / time.Duration(
				int(state.sampleHz)/state.sampleSize))
	}

	var tAvg, _ = state.durWindow.Update(now.Sub(state.prevTime).Seconds())

	var peak = peakElement(dest, count)
	if peak <= 0 {
		return
	}

	var scaleAvg, scaleSd = state.scaleWindow.Update(peak)

	var multi = math.Max(scaleAvg+(2*scaleSd), 1)

	n2s3SecondPass(dest, count, multi, state)

	for xBin := 0; xBin < count; xBin++ {
		dest[xBin] = state.prevBins[xBin].addZero(
			n2s3Delta(state, xBin, tAvg, dest[xBin], state.prevBins[xBin].value))
	}

	state.prevTime = time.Now()
}

func peakElement(bins []float64, count int) (max float64) {
	for xBin := 0; xBin < count; xBin++ {
		if max < bins[xBin] {
			max = bins[xBin]
		}
	}

	return
}

func n2s3SecondPass(bins []float64, count int, mult float64, state *State) {
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
		bins[xBin] = math.Min(1, bins[xBin]/mult)

		sum += bins[xBin]

		if bins[xBin] > max {
			max = bins[xBin]
		}

		if bins[xBin] < min {
			min = bins[xBin]
		}

		var d = bins[xBin] - state.prevBins[xBin].value
		sumDelta += d

		if d > maxDelta {
			maxDelta = d
		}

		if d < minDelta {
			minDelta = d
		}

		var ad = math.Abs(d)
		sumAbsDelta += ad

		if ad > maxAbsDelta {
			maxAbsDelta = ad
		}

		if ad < minAbsDelta {
			minAbsDelta = ad
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

func n2s3Delta(state *State, i int, t, r, p float64) float64 {
	var d, _, _, _, _, _ = n2s3DeltaHelper(state.sumAbsDelta.value, r, p)

	// use the ratio of change / peak height of transition
	// as percent value

	// var absEnergy = math.Abs(energy)

	// percentage of activity this bar counts for

	// if pct > 0.1 && pct < 1 {
	// return d * pct * math.Pow(2*pct, 0.5)
	// }

	// var mdp = ad / state.avgAbsDelta.value

	// if r < p {
	// 	return d * math.Max(0.000000001, math.Min(1, math.Pow(pct, 60*t)))
	// }

	if p == 0 {
		return d
	}

	return d * math.Log10(d)
}

func n2s3DeltaHelper(a, r, p float64) (d, ad, dp, max, min, pct float64) {
	d = r - p
	ad = math.Abs(d)
	dp = ad / a
	max = math.Max(r, p)
	min = max - ad
	pct = 1 - math.Max(0.05, math.Min(1, ad/max))

	return
}
