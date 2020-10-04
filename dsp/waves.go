package dsp

import "math"

// Waves comes from cava
// https://github.com/karlstav/cava/blob/master/cava.c#L144
func Waves(factor float64, dSet *DataSet) {

	for xPass := 0; xPass < dSet.numBins; xPass++ {
		dSet.binBuf[xPass] /= factor

		for xBin := xPass - 1; xBin >= 0; xBin-- {
			dSet.binBuf[xBin] = math.Max(dSet.binBuf[xPass]-
				math.Pow(float64(xPass-xBin), 2), dSet.binBuf[xBin])
		}

		for xBin := xPass + 1; xBin < dSet.numBins; xBin++ {
			dSet.binBuf[xBin] = math.Max(dSet.binBuf[xPass]-
				math.Pow(float64(xBin-xPass), 2), dSet.binBuf[xBin])
		}
	}
}
