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

// Buffer describes an object that can have `Ptr` called on it.
// it should be a slice or other object that would be contiguous in C
type Buffer interface {
	Ptr(...int) unsafe.Pointer
}

// CmplxType is a type used for FFTW complex
// it should be `complex128` but we can play with it
type CmplxType = complex128

// CmplxBuffer is a slice of CmplxType
type CmplxBuffer []CmplxType

// Ptr returns a pointer for use with CGO
func (cb CmplxBuffer) Ptr(n ...int) unsafe.Pointer {
	if len(n) > 0 {
		return unsafe.Pointer(&cb[n[0]])
	}

	return unsafe.Pointer(&cb[0])
}

// Direction is an FFTW direction of operation flag
type Direction int

const (
	// Forward means go from seconds to frequency time
	Forward Direction = C.FFTW_FORWARD
	// Backward meads go from frequence time to seconds
	Backward Direction = C.FFTW_BACKWARD
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
func New(in, out Buffer, d0, d1 int, dir Direction, flag Flag) *Plan {
	var (
		inC   = (*C.fftw_complex)(in.Ptr())
		outC  = (*C.fftw_complex)(out.Ptr())
		d0C   = C.int(d0)
		d1C   = C.int(d1)
		dirC  = C.int(dir)
		flagC = C.uint(flag)
	)
	p := C.fftw_plan_dft_2d(d0C, d1C, inC, outC, dirC, flagC)
	return &Plan{p}
}
