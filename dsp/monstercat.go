package dsp

import "math"

// Monstercat does monstercat smoothing
// https://github.com/karlstav/cava/blob/master/cava.c#L157
func Monstercat(factor float64, ds *DataSet) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var lf = math.Log(factor)

	for xBin := 1; xBin < ds.numBins; xBin++ {

		for xTrgt := 0; xTrgt < ds.numBins; xTrgt++ {

			if xBin != xTrgt {
				var tmp = ds.binBuf[xBin] / math.Exp(lf*math.Abs(float64(xBin-xTrgt)))

				if tmp > ds.binBuf[xTrgt] {
					ds.binBuf[xTrgt] = tmp
				}
			}
		}
	}
}
