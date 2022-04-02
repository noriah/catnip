//go:build cgo && !windows && !nofftw

package fft

// This only included bindings are those that are needed by catnip.
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

// FFTW is true if Catnip is built with cgo.
const FFTW = true

// Plan holds an FFTW C plan
type Plan struct {
	input  []float64
	output []complex128
	cPlan  C.fftw_plan
}

// Init sets up the plan so we dont run checks during execute
func (p *Plan) init() {
	if p.cPlan == nil {
		p.cPlan = C.fftw_plan_dft_r2c_1d(
			C.int(len(p.input)),
			(*C.double)(unsafe.Pointer(&p.input[0])),
			(*C.fftw_complex)(unsafe.Pointer(&p.output[0])),
			C.FFTW_MEASURE,
		)

		runtime.SetFinalizer(p, (*Plan).destroy)
	}
}

// Execute runs the plan
func (p *Plan) Execute() {
	C.fftw_execute(p.cPlan)
}

// destroy releases resources
func (p *Plan) destroy() {
	C.fftw_destroy_plan(p.cPlan)
}
