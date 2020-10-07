package dsp

import (
	"github.com/noriah/tavis/fft"
	"github.com/noriah/tavis/util"
)

// DataSet represents a channel or sample index in a series frame
type DataSet struct {
	id int

	sampleSize int
	sampleHz   float64

	inputBuf  []float64
	inputSize int

	fftBuf  []complex128
	fftSize int

	fftPlan *fft.Plan

	binBuf  []float64
	numBins int

	prevBuf []float64

	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow
}

// ID returns the set id
func (ds *DataSet) ID() int {
	return ds.id
}

// Input returns the buffer we read from
func (ds *DataSet) Input() []float64 {
	return ds.inputBuf
}

// Bins returns the bins that we have as a silce
func (ds *DataSet) Bins() []float64 {
	return ds.binBuf[0:ds.numBins:ds.numBins]
}

// Len returns the number of bins we have processed
func (ds *DataSet) Len() int {
	return ds.numBins
}

// ExecuteFFTW executes fftw math on the source buffer
func (ds *DataSet) ExecuteFFTW() {
	ds.fftPlan.Execute()
}

// Props returns the samle hz and size
func (ds *DataSet) Props() (float64, int) {
	return ds.sampleHz, ds.sampleSize
}

