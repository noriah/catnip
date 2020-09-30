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

// RealType is an alias for us
type RealType = float64

// ComplexType is an alias for us
type ComplexType = complex128

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
func New(in []RealType, out []ComplexType, d0, d1 int, flag Flag) *Plan {
	var (
		inC   = (*C.double)(unsafe.Pointer(&in[0]))
		outC  = (*C.fftw_complex)(unsafe.Pointer(&out[0]))
		d0C   = C.int(d0)
		d1C   = C.int(d1)
		flagC = C.uint(flag)
	)
	p := C.fftw_plan_dft_r2c_2d(d0C, d1C, inC, outC, flagC)
	return &Plan{p}
}
