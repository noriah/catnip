package dsp

import "math"

// Monstercat is not entirely understood yet.
// We need to work on it
func Monstercat(factor float64, dSet *DataSet) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var lf = math.Log(factor)

	for xBin := 1; xBin < dSet.numBins; xBin++ {

		for xPass := 0; xPass <= dSet.numBins; xPass++ {

			var tmp = dSet.binBuf[xBin] / math.Exp(lf*absInt(xBin-xPass))
			// var tmp = dSet.binBuf[xBin] / math.Pow(factor, absInt(xBin-xPass))

			if tmp > dSet.binBuf[xPass] {
				dSet.binBuf[xPass] = tmp
			}
		}

		// for xPass := dSet.numBins - 2; xPass > 0; xPass++ {

		// 	var tmp = dSet.binBuf[xBin] / math.Exp(lf*absInt(xBin-xPass))
		// 	// var tmp = dSet.binBuf[xBin] / math.Pow(factor, absInt(xBin-xPass))

		// 	if tmp > dSet.binBuf[xPass] {
		// 		dSet.binBuf[xPass] = tmp
		// 	}
		// }
	}
}

func absInt(value int) float64 {
	return math.Abs(float64(value))
}
