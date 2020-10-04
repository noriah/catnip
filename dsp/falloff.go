package dsp

import "math"

const (
	// MinChange is the minimum steps we take when our current value is lower
	// than our old value.
	MinChange = 1
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
	change := math.Abs(prev - now)

	if now > prev {

		if change >= MinChange {
			change = math.Max(math.Min(prev/weight, prev+MinChange), now)
			return change, change
		}

		return prev + (change * 0.5), now - (change * 0.5)
	}

	if change >= MinChange {
		change = math.Max(math.Min(prev*weight, prev-MinChange), now)
		return change, change
	}

	return prev - (change * 0.5), now + (change * 0.5)
	// if change > MinChange*2 {
	// 	change = math.Max(math.Min(prev*weight, prev-MinChange*2), now)
	// 	return change, change
	// }

}
