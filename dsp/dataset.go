package dsp

import (
	"github.com/noriah/tavis/fft"
	"github.com/noriah/tavis/util"
)

// DataSet represents a channel or sample index in a series frame
type DataSet struct {
	id int

	spectrum *Spectrum

	fftPlan *fft.Plan

	dataBuf  []complex128
	dataSize int

	binBuf  []float64
	prevBuf []float64
	numBins int

	peakHeight float64

	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow
}

// ID returns the set id
func (ds *DataSet) ID() int {
	return ds.id
}

// Bins returns the bins that we have as a silce
func (ds *DataSet) Bins() []float64 {
	return ds.binBuf[:ds.numBins]
}

// Size returns the number of bins we have processed
func (ds *DataSet) Size() int {
	return ds.numBins
}

// ExecuteFFTW executes fftw math on the source buffer
func (ds *DataSet) ExecuteFFTW() {
	ds.fftPlan.Execute()
}
