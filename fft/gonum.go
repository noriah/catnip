//go:build !cgo || nofftw || (windows && !fftwonwin)

package fft

import "gonum.org/v1/gonum/dsp/fourier"

// FFTW is false if Catnip is not built with cgo. It will use gonum instead.
const FFTW = false

// Plan holds a gonum FFT plan.
type Plan struct {
	input  []float64
	output []complex128
	fft    *fourier.FFT
}

// Init sets up the plan so we dont run checks during execute
func (p *Plan) init() {
	if p.fft == nil {
		p.fft = fourier.NewFFT(len(p.input))
	}
}

// Execute executes the gonum plan.
func (p *Plan) Execute() {
	p.fft.Coefficients(p.output, p.input)
}
