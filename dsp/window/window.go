// Package window provides Window Functions for singnal analysis
//
// See https://wikipedia.org/wiki/Window_function
package window

import "math"

// Function is a function that will do window things for you on a slice
type Function func(buf []float64)

// Rectangle is just do nothing
func Rectangle() Function {
	return func(buf []float64) {
		// do nothing
	}
}

// CosSum modifies the buffer to conform to a cosine sum window following a0
func CosSum(a0 float64) Function {
	return func(buf []float64) {
		size := len(buf)
		a1 := 1.0 - a0
		coef := 2.0 * math.Pi / float64(size)
		for n := 0; n < size; n++ {
			buf[n] *= (a0 - (a1 * math.Cos(coef*float64(n))))
		}
	}
}

// sinc(x) = sin(pi * x) / (pi * x)
func sinc(x float64) float64 {
	if x == 0.0 {
		return 0.0
	}
	piX := math.Pi * x
	return math.Sin(piX) / piX
}

// Lanczos modifies the buffer to a Lanczos window
//
// w[n] = sinc((2n / N) - 1)
//
// N = size
// n = element
// k = 2 / N
//
// buf[n] = sinc(kn - 1)
//
// https://www.wikiwand.com/en/Window_function#/Other_windows
func Lanczos() Function {
	return func(buf []float64) {
		k := 2.0 / float64(len(buf))
		for n := range buf {
			buf[n] *= sinc((k * float64(n)) - 1.0)
		}
	}
}

// HammingConst is the hamming window constant
const HammingConst = 25.0 / 46.0

// Hamming modifies the buffer to a Hamming window
func Hamming() Function {
	return CosSum(HammingConst)
}

// Hann modifies the buffer to a Hann window
func Hann() Function {
	return CosSum(0.5)
}

// Bartlett modifies the buffer to a Bartlett window
func Bartlett() Function {
	return func(buf []float64) {
		N := float64(len(buf))
		for n := range buf {
			buf[n] *= (1.0 - (math.Abs(((2.0 * float64(n)) - N) / N)))
		}
	}
}

// Blackman modifies the buffer to a Blackman window
//
// N = size
// n = element
// a = 0.16
// a_0 = (1 - a) / 2
// a_1 = 1 / 2
// a_2 = a / 2
// w[n] = a_0 - a_1 * cos((2 * pi * n) / N) + a_2 * cos((4 * pi * n) / N)
func Blackman() Function {
	return func(buf []float64) {
		N := float64(len(buf))
		twoPi := 2.0 * math.Pi

		for n := range buf {
			twoPiX := twoPi * (float64(n) / N)
			buf[n] *= 0.42 - (0.5 * math.Cos(twoPiX)) + (0.08 * math.Cos(2.0*twoPiX))
		}
	}
}

// PlanckTaper modifies the buffer to a Planck-taper window
//
// not sure how i got this
func PlanckTaper(e float64) Function {
	return func(buf []float64) {
		size := len(buf)
		eN := e * float64(size)

		buf[0] *= 0
		for n := 1; n < int(eN); n++ {
			buf[n] *= 1.0 / (1.0 + math.Exp((eN/float64(n))-(eN/(eN-float64(n)))))
		}

		for n := 1; n <= size/2; n++ {
			buf[size-n] *= buf[n-1]
		}
	}
}
