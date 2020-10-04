package dsp

import "math"

// Monstercat is not entirely understood yet.
// https://github.com/karlstav/cava/blob/master/cava.c#L157
func Monstercat(factor float64, dSet *DataSet) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var lf = math.Log(factor)

	for xPass := 0; xPass < dSet.numBins; xPass++ {

		for xBin := xPass - 1; xBin >= 0; xBin-- {

			var tmp = dSet.binBuf[xPass] / math.Exp(lf*float64(xPass-xBin))

			if tmp > dSet.binBuf[xBin] {
				dSet.binBuf[xBin] = tmp
			}
		}

		for xBin := dSet.numBins + 1; xBin < dSet.numBins; xBin++ {
			var tmp = dSet.binBuf[xPass] / math.Exp(lf*float64(xBin-xPass))
			if tmp > dSet.binBuf[xBin] {
				dSet.binBuf[xBin] = tmp
			}
		}
	}
}

func absInt(value int) float64 {
	return math.Abs(float64(value))
}
