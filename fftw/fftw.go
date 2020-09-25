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
	fftw_p C.fftw_plan
}

func (p *Plan) Execute() {
	C.fftw_execute(p.fftw_p)
}

func (p *Plan) Destroy() {
	C.fftw_destroy_plan(p.fftw_p)
}

func New(in []float32, out []FftwComplexType, d0, d1 int, dir Direction, flag Flag) *Plan {
	var (
		inC   = (*C.fftw_complex)(unsafe.Pointer(&in[0]))
		outC  = (*C.fftw_complex)(unsafe.Pointer(&out[0]))
		d0C   = C.int(d0)
		d1C   = C.int(d1)
		dirC  = C.int(dir)
		flagC = C.uint(flag)
	)
	p := C.fftw_plan_dft_2d(d0C, d1C, inC, outC, dirC, flagC)
	return &Plan{p}
}
