package dsp

import "math"

// Waves comes from cava
// https://github.com/karlstav/cava/blob/master/cava.c#L144
func Waves(buf []float64, count int, factor float64) {

	for xBin := 1; xBin < count; xBin++ {
		buf[xBin] /= factor

		for xTarget := xBin - 1; xTarget > 0; xTarget-- {
			buf[xTarget] = math.Max(buf[xBin]-
				math.Pow(float64(xBin-xTarget), 2), buf[xTarget])
		}

		for xTarget := xBin + 1; xTarget <= count; xTarget++ {
			buf[xTarget] = math.Max(
				buf[xBin]-math.Pow(float64(xTarget-xBin), 2), buf[xTarget])
		}
	}
}
