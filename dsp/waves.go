package dsp

import (
	"math"
)

// Waves comes from cava
// https://github.com/karlstav/cava/blob/master/cava.c#L144
func Waves(factor float64, ds *DataSet) {

	for xPass := 0; xPass < ds.numBins; xPass++ {
		ds.binBuf[xPass] /= factor

		for xBin := xPass - 1; xBin >= 0; xBin-- {
			ds.binBuf[xBin] = math.Max(ds.binBuf[xPass]-
				math.Pow(float64(xPass-xBin), 2), ds.binBuf[xBin])
		}

		for xBin := xPass + 1; xBin < ds.numBins; xBin++ {
			ds.binBuf[xBin] = math.Max(ds.binBuf[xPass]-
				math.Pow(float64(xBin-xPass), 2), ds.binBuf[xBin])
		}
	}
}
