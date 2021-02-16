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

// Init sets up the plan so we dont run checks during execute
func (p *Plan) Init() {
	if p.fft == nil {
		p.fft = fourier.NewFFT(len(p.Input))
	}
}

// Execute executes the gonum plan.
func (p *Plan) Execute() {
	p.fft.Coefficients(p.Output, p.Input)
}
