package dsp

import "github.com/noriah/tavis/fft"

// BinSet represents a stream of data
type BinSet struct {
	count  int
	buffer []float64
	plan   *fft.Plan
}

// Bins returns the bins that we have as a silce
func (bs *BinSet) Bins() []float64 {
	return bs.buffer
}

// Len returns the number of bins we have processed
func (bs *BinSet) Len() int {
	return bs.count
}
