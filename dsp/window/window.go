// Package window provides Window Functions for singnal analysis
//
// See https://wikipedia.org/wiki/Window_function
package window

import "math"

// CosSum modifies the buffer to conform to a cosine sum window following a0
func CosSum(buf []float64, size int, a0 float64) {
	var a1 = 1.0 - a0
	var coef = 2 * math.Pi / float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (a0 - a1*math.Cos(coef*float64(n)))
	}
}

// Hamming modifies the buffer to a Hamming window
func Hamming(buf []float64, size int) {
	CosSum(buf, size, 0.53836)
}

// Hann modifies the buffer to a Hann window
func Hann(buf []float64, size int) {
	CosSum(buf, size, 0.5)
}

// Bartlett modifies the buffer to a Bartlett window
func Bartlett(buf []float64, size int) {
	var fSize = float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (1.0 - math.Abs((2.0*float64(n)-fSize)/fSize))
	}
}

// Blackman modifies the buffer to a Blackman window
func Blackman(buf []float64, size int) {
	var fSize = float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (0.42 - (0.5 * math.Cos(((2.0 * float64(n)) / fSize))) +
			(0.08 * math.Cos((4.0 * float64(n))) / fSize))
	}
}
