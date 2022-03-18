// Package fft provides generic abstractions around fourier transformers.
package fft

func InitPlan(pointer **Plan, input []float64, output []complex128) {
	(*pointer) = &Plan{
		input:  input,
		output: output,
	}

	(*pointer).init()
}
