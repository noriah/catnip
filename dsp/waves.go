package dsp

import "math"

// Waves comes from cava
// https://github.com/karlstav/cava/blob/master/cava.c#L144
func Waves(buf []float64, count int, factor float64) {

	for xPass := 0; xPass < count; xPass++ {
		buf[xPass] /= factor

		for xBin := xPass - 1; xBin >= 0; xBin-- {
			buf[xBin] = math.Max(buf[xPass]-
				math.Pow(float64(xPass-xBin), 2), buf[xBin])
		}

		for xBin := xPass + 1; xBin < count; xBin++ {
			buf[xBin] = math.Max(buf[xPass]-
				math.Pow(float64(xBin-xPass), 2), buf[xBin])
		}
	}
}
