// +build cgo

package fft

// This only included bindings are those that are needed by tavis.
// This includes the use of `fftw_plan_dft_r2c_2d`.
// It is the only fftw plan we need, and the only one we have chosen to
// implement here.

// #cgo pkg-config: fftw3
// #include <fftw3.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// FFTW is true if Tavis is built with cgo.
const FFTW = true

// Plan holds an FFTW C plan
type Plan struct {
	input  []float64
	output []complex128
	cPlan  C.fftw_plan
}

// Execute runs the plan
func (p *Plan) Execute() {
	C.fftw_execute(p.cPlan)
}

// destroy releases resources
func (p *Plan) destroy() {
	C.fftw_destroy_plan(p.cPlan)
}

// NewPlan returns a new FFTW Plan for use with FFTW
func NewPlan(in []float64, out []complex128) *Plan {
	var plan = &Plan{
		input:  in,
		output: out,
		cPlan: C.fftw_plan_dft_r2c_1d(
			C.int(len(in)),
			(*C.double)(unsafe.Pointer(&in[0])),
			(*C.fftw_complex)(unsafe.Pointer(&out[0])),
			C.FFTW_MEASURE,
		),
	}

	// Rely on the runtime to free memory.
	runtime.SetFinalizer(plan, (*Plan).destroy)

	return plan
}
