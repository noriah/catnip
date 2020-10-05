package dsp

import "math"

const (
	// MinFall is the minimum steps we take when our current value is lower
	// than our old value.
	MinFall = 1
)

// Falloff does falling off things
func Falloff(weight float64, dSet *DataSet) {
	weight = math.Max(0.7, math.Min(1, weight))

	for xBin := 0; xBin <= dSet.numBins; xBin++ {
		mag := falloff(weight, dSet.prevBuf[xBin], dSet.binBuf[xBin])
		dSet.binBuf[xBin] = mag
		dSet.prevBuf[xBin] = mag
	}
}

func falloff(weight, prev, now float64) float64 {
	return math.Max(math.Min(prev*weight, prev-MinFall), now)
}
