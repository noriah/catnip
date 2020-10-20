package dsp

import "math"

// Monstercat does monstercat "smoothing"
// https://github.com/karlstav/cava/blob/master/cava.c#L157
func Monstercat(bins []float64, count int, factor float64) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var lf = math.Log(factor)

	for xBin := 1; xBin < count; xBin++ {

		for xTrgt := 0; xTrgt < count; xTrgt++ {

			if xBin != xTrgt {
				var tmp = bins[xBin] / math.Exp(lf*math.Abs(float64(xBin-xTrgt)))

				if tmp > bins[xTrgt] {
					bins[xTrgt] = tmp
				}
			}
		}
	}
}
