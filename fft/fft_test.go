package fft

import "testing"

func Benchmark(b *testing.B) {
	if FFTW {
		b.Log("Benchmarking FFTW.")
	} else {
		b.Log("Benchmarking gonum (built without cgo).")
	}

	reals := generateReals()
	cmplx := make([]complex128, len(reals)/2+1)
	fftpl := Plan{
		input:  reals,
		output: cmplx,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fftpl.Execute()
	}
}

// Adapted from https://github.com/project-gemmi/benchmarking-fft/blob/master/1d-r.cpp

const numReals = 44100

func generateReals() []float64 {
	input := make([]float64, numReals)

	c := 3.1
	for i := range input {
		c += 0.3
		input[i] = 2*c - c*c
	}

	return input
}
