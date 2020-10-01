package dsp

import (
	"math"

	"github.com/noriah/tavis/dsp/n2s3"
	"github.com/noriah/tavis/fft"
)

// Spectrum Constants
const (
	MaxBins = 1024
)

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBins int
	numBins int

	setCount int

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	loCuts []int
	hiCuts []int
}

// NewSpectrum will set up our spectrum
func NewSpectrum(rate float64, size int) *Spectrum {

	var sp = &Spectrum{
		maxBins:    MaxBins,
		sampleSize: size,
		sampleRate: rate,
	}

	sp.loCuts = make([]int, sp.maxBins+1)
	sp.hiCuts = make([]int, sp.maxBins+1)

	sp.Recalculate(sp.maxBins, 20, sp.sampleRate/2)

	return sp
}

// DataSet reurns a new data set with settings matching this spectrum
func (sp *Spectrum) DataSet(input []float64) *DataSet {

	if input == nil {
		input = make([]float64, sp.sampleSize)
	}

	var fftSize = (sp.sampleSize / 2) + 1

	var set = &DataSet{
		id:        sp.setCount,
		inputBuf:  input,
		inputSize: len(input),
		fftSize:   fftSize,
		fftBuf:    make([]complex128, fftSize),
		binBuf:    make([]float64, sp.maxBins),
		prevBuf:   make([]float64, sp.maxBins),

		sampleHz:   sp.sampleRate,
		sampleSize: sp.sampleSize,

		N2S3State:  n2s3.NewState(sp.sampleRate, sp.sampleSize, sp.maxBins),
		ScaleState: NewScaleState(sp.sampleRate, sp.sampleSize),
	}

	set.fftPlan = fft.NewPlan(set.inputBuf, set.fftBuf, sp.sampleSize)

	sp.setCount++

	return set
}

// Recalculate rebuilds our frequency bins with bins bin counts
//
// reference: https://github.com/karlstav/cava/blob/master/cava.c#L654
// reference: https://github.com/noriah/cli-visualizer/blob/master/src/Transformer/SpectrumTransformer.cpp#L598
func (sp *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > sp.maxBins {
		bins = sp.maxBins
	}

	sp.numBins = bins

	var cBins = float64(bins + 1)

	var cFreq = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin := 0; xBin <= bins; xBin++ {
		// Fix issue where recalculations may not be accurate due to
		// previous runs
		sp.loCuts[xBin] = 0
		sp.hiCuts[xBin] = 0

		vFreq := (((float64(xBin+1) / cBins) - 1) * cFreq)
		vFreq = hi * math.Pow(10.0, vFreq)
		vFreq = (vFreq / (sp.sampleRate / 2)) * (float64(sp.sampleSize) / 4)

		sp.loCuts[xBin] = int(math.Floor(vFreq))

		if xBin > 0 {
			if sp.loCuts[xBin] <= sp.loCuts[xBin-1] {
				sp.loCuts[xBin] = sp.loCuts[xBin-1] + 1
			}

			sp.hiCuts[xBin-1] = sp.loCuts[xBin-1]
		}
	}

	return sp.numBins
}

// Generate makes numBins and dumps them in the buffer
func (sp *Spectrum) Generate(ds *DataSet) {

	ds.ExecuteFFTW()

	ds.numBins = sp.numBins

	for xBin := 0; xBin < ds.numBins; xBin++ {

		var vM = 0.0

		for xF := sp.loCuts[xBin]; xF <= sp.hiCuts[xBin] &&
			xF >= 0 &&
			xF < ds.fftSize; xF++ {

			vM += pyt(ds.fftBuf[xF])
		}

		vM /= float64(sp.hiCuts[xBin] - sp.loCuts[xBin] + 1)

		vM *= (math.Log2(float64(2+xBin)) * (100.0 / float64(ds.numBins)))

		ds.binBuf[xBin] = math.Pow(vM, 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}
