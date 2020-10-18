// +build !cgo

package fft

import (
	"gonum.org/v1/gonum/dsp/fourier"
)

// FFTW is false if Catnip is not built with cgo. It will use gonum instead.
const FFTW = false

// Plan holds a gonum FFT plan.
type Plan struct {
	Input  []float64
	Output []complex128
	fft    *fourier.FFT
}

// Execute executes the gonum plan.
func (p *Plan) Execute() {
	if p.fft == nil {
		p.fft = fourier.NewFFT(len(p.Input))
	}
	p.fft.Coefficients(p.Output, p.Input)
}
