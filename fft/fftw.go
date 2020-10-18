// +build ignore

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
	Input  []float64
	Output []complex128
	cPlan  C.fftw_plan
}

// Execute runs the plan
func (p *Plan) Execute() {
	if p.cPlan == nil {
		p.cPlan = C.fftw_plan_dft_r2c_1d(
			C.int(len(p.Input)),
			(*C.double)(unsafe.Pointer(&p.Input[0])),
			(*C.fftw_complex)(unsafe.Pointer(&p.Output[0])),
			C.FFTW_MEASURE,
		)

		runtime.SetFinalizer(p, (*Plan).destroy)
	}

	C.fftw_execute(p.cPlan)
}

// destroy releases resources
func (p *Plan) destroy() {
	C.fftw_destroy_plan(p.cPlan)
}
