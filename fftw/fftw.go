package fftw

// #include <fftw3.h>
// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lfftw3 -lm
import "C"

import "unsafe"

type FftwComplexType = float64

type Direction int

const (
	Forward  = Direction(C.FFTW_FORWARD)
	Backward = Direction(C.FFTW_BACKWARD)
)

type Flag uint

const (
	Estimate = Flag(C.FFTW_ESTIMATE)
	Measure  = Flag(C.FFTW_MEASURE)
)

type Plan struct {
	cPlan C.fftw_plan
}

func (p *Plan) Execute() {
	C.fftw_execute(p.cPlan)
}

func (p *Plan) Destroy() {
	C.fftw_destroy_plan(p.cPlan)
}

func New(in []float32, out []FftwComplexType, d0, d1 int, flag Flag) *Plan {
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
