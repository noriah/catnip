// Package fftw contains specific Go bindings for the FFTW C library
//
// The only included bindings are those that are needed by tavis.
// This includes the use of `fftw_plan_dft_r2c_2d`.
// It is the only fftw plan we need, and the only one we have chosen to
// implement here.
package fftw

// #cgo pkg-config: fftw3
// #include <fftw3.h>
import "C"

import (
	"unsafe"
)

// Flag is an FFTW method flag
type Flag uint

const (
	// Estimate is C.FFTW_ESTIMATE
	Estimate Flag = C.FFTW_ESTIMATE
	// Measure is C.FFTW_MEASURE
	Measure Flag = C.FFTW_MEASURE
)

// Plan holds an FFTW C plan
type Plan struct {
	cPlan C.fftw_plan
}

// Execute runs the plan
func (p *Plan) Execute() {
	C.fftw_execute(p.cPlan)
}

// Destroy releases resources
func (p *Plan) Destroy() {
	C.fftw_destroy_plan(p.cPlan)
}

// New returns a new FFTW Plan for use with FFTW
func New(in []float64, out []complex128, d0 int, flag Flag) *Plan {
	return &Plan{C.fftw_plan_dft_r2c_1d(
		C.int(d0),
		(*C.double)(unsafe.Pointer(&in[0])),
		(*C.fftw_complex)(unsafe.Pointer(&out[0])),
		C.uint(flag),
	)}
}
