package dsp

import "math"

const (
	// MinDelta is the minimum steps we take when our current value is lower
	// than our old value.
	MinDelta = 1
)

// Falloff does falling off things
func Falloff(weight float64, ds *DataSet) {
	weight = math.Max(0.7, math.Min(1, weight))

	for xBin := 0; xBin <= ds.numBins; xBin++ {
		ds.binBuf[xBin], ds.prevBuf[xBin] = falloff(
			weight,
			ds.prevBuf[xBin],
			ds.binBuf[xBin],
		)
	}
}

func falloff(weight, prev, now float64) (float64, float64) {

	delta := math.Abs(prev - now)

	if now >= prev {

		if delta >= MinDelta {

			return prev + (delta * 0.5), now - (delta * 0.5)
		}

		delta = math.Max(math.Min(prev/weight, prev+MinDelta), now)
		return delta, delta
	}

	if delta >= MinDelta {

		delta = math.Max(math.Min(prev*weight, prev-MinDelta), now)
		return delta, delta
	}

	return prev - (delta * 0.5), now + (delta * 0.5)
}
