package dsp

import (
	"math"
)

const (
	// MinDelta is the minimum steps we take when our current value is lower
	// than our old value.
	MinDelta = 0.1

	MajorDelta = 0.2

	MinCharDelta = 1 / 9
)

// Falloff does smoothing off things
func Falloff(weight float64, ds *DataSet) {
	for xBin := 0; xBin <= ds.Len(); xBin++ {
		ds.binBuf[xBin], ds.prevBuf[xBin] = zeroPlus(falloff(
			weight,
			ds.binBuf[xBin],
			ds.prevBuf[xBin],
		))
	}
}

func falloff(weight, now, prev float64) (float64, float64) {

	var delta = math.Abs(now - prev)

	if delta < MinDelta {
		return prev, now
	}

	if now > prev {
		if delta >= MajorDelta {
			delta = math.Max(now, math.Min(prev+(delta*weight), prev+MinDelta))
			return delta, delta
		}

		if delta >= MinDelta {
			return prev + (delta * 0.5), prev + (delta * 0.5)
		}

		return prev + (delta * 0.5), prev + (delta * 0.5)
	}

	var last = now + (delta * 0.)

	if delta >= MajorDelta {
		delta = math.Max(now, math.Min(prev-(delta*weight), prev-MinDelta))
		return delta, last
	}

	if delta >= MinDelta {
		return prev - (delta * 0.75), now + (delta * 0.125)
	}

	return prev - (delta * 0.5), last
}

func zeroPlus(a, b float64) (float64, float64) {
	return math.Max(0, a), math.Max(0, b)
}
