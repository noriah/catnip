// Package window provides Window Functions for singnal analysis
//
// See https://wikipedia.org/wiki/Window_function
package window

import "math"

// Function is a function that will do window things for you
type Function func(buf []float64)

// Rectangle is just do nothing
func Rectangle(buf []float64) {
	// do nothing
}

// CosSum modifies the buffer to conform to a cosine sum window following a0
func CosSum(buf []float64, a0 float64) {
	var size = len(buf)
	var a1 = 1.0 - a0
	var coef = 2.0 * math.Pi / float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (a0 - a1*math.Cos(coef*float64(n)))
	}
}

// Hamming modifies the buffer to a Hamming window
func Hamming(buf []float64) {
	CosSum(buf, 25.0/46.0)
}

// Hann modifies the buffer to a Hann window
func Hann(buf []float64) {
	CosSum(buf, 0.5)
}

// Bartlett modifies the buffer to a Bartlett window
func Bartlett(buf []float64) {
	var size = len(buf)
	var fSize = float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (1.0 - math.Abs((2.0*float64(n)-fSize)/fSize))
	}
}

// Blackman modifies the buffer to a Blackman window
func Blackman(buf []float64) {
	var size = len(buf)
	var fSize = float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (0.42 - (0.5 * math.Cos(((2.0 * float64(n)) / fSize))) +
			(0.08 * math.Cos((4.0 * float64(n))) / fSize))
	}
}

// PlanckTaper modifies the buffer to a Planck-taper window
func PlanckTaper(buf []float64, e float64) {
	var size = len(buf)
	var eN = e * float64(size)

	buf[0] *= 0
	for n := 1; n < int(eN); n++ {
		buf[n] *= 1.0 / (1.0 + math.Exp((eN/float64(n))-(eN/(eN-float64(n)))))
	}

	for n := 1; n <= size/2; n++ {
		buf[size-n] *= buf[n-1]
	}
}
