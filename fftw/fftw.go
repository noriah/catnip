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
	cPlans []C.fftw_plan
}

// Execute runs the plan
func (p *Plan) Execute() {
	for xID := 0; xID < len(p.cPlans); xID++ {
		C.fftw_execute(p.cPlans[xID])
	}
}

// Destroy releases resources
func (p *Plan) Destroy() {
	for xID := 0; xID < len(p.cPlans); xID++ {
		C.fftw_destroy_plan(p.cPlans[xID])
	}
}

// New returns a new FFTW Plan for use with FFTW
func New(in []float64, out []complex128, d0, d1 int, flag Flag) *Plan {

	var (
		d1C   = C.int(d1)
		flagC = C.uint(flag)
		plan  = &Plan{make([]C.fftw_plan, d0)}
	)

	for xID := 0; xID < d0; xID++ {
		plan.cPlans[xID] = C.fftw_plan_dft_r2c_1d(
			d1C,
			(*C.double)(unsafe.Pointer(&in[d1*xID])),
			(*C.fftw_complex)(unsafe.Pointer(&out[((d1/2)+1)*xID])),
			flagC,
		)
	}

	return plan
}
