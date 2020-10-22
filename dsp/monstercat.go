package dsp

import "math"

// Monstercat does monstercat "smoothing"
//
// https://github.com/karlstav/cava/blob/master/cava.c#L157
//
// TODO(winter): make faster (rewrite)
//	slow and hungry as heck!
//	lets look into SIMD
//
//  lets look
func Monstercat(bins []float64, count int, factor float64) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var vFactP = math.Log(factor)

	for xBin := 1; xBin < count; xBin++ {

		for xTrgt := 0; xTrgt < count; xTrgt++ {

			if xBin != xTrgt {
				var tmp = bins[xBin]
				tmp /= math.Exp(vFactP * math.Abs(float64(xBin-xTrgt)))

				if tmp > bins[xTrgt] {
					bins[xTrgt] = tmp
				}
			}
		}
	}
}
