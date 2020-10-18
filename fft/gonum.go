// +build !cgo

package fft

import "gonum.org/v1/gonum/dsp/fourier"

// FFTW is false if Tavis is not built with cgo. It will use gonum instead.
const FFTW = false

// Plan holds a gonum FFT plan.
type Plan struct {
	input  []float64
	output []complex128
	fft    *fourier.FFT
}

// NewPlan creates a new gonum plan.
func NewPlan(in []float64, out []complex128, n int) *Plan {
	return &Plan{
		input:  in,
		output: out,
		fourier.NewFFT(n),
	}
}

// Execute executes the gonum plan.
func (p Plan) Execute() {
	p.fft.Coefficients(p.output, p.input)
}
