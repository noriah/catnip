package dsp

import (
	"math"

	"github.com/noriah/tavis/fft"
	"github.com/noriah/tavis/util"
)

// Spectrum Constants
const (
	// ScalingFastWindow in seconds
	ScalingSlowWindow = 10

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.1

	MaxBins = 1024
)

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBins int
	numBins int

	setCount int

	dataSize int

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	loCuts []int
	hiCuts []int
}

// NewSpectrum will set up our spectrum
func NewSpectrum(rate float64, size int) *Spectrum {

	var s = &Spectrum{
		maxBins:    MaxBins,
		dataSize:   (size / 2) + 1,
		sampleSize: size,
		sampleRate: rate,
	}

	s.loCuts = make([]int, s.maxBins+1)
	s.hiCuts = make([]int, s.maxBins+1)

	s.Recalculate(s.maxBins, 20, s.sampleRate/2)

	return s
}

// DataSet reurns a new data set with settings matching this spectrum
func (s *Spectrum) DataSet(input []float64) *DataSet {

	slowMax := int((ScalingSlowWindow*s.sampleRate)/float64(s.sampleSize)) * 2
	fastMax := int((ScalingFastWindow*s.sampleRate)/float64(s.sampleSize)) * 2

	var set = &DataSet{
		id:         s.setCount,
		dataSize:   s.dataSize,
		dataBuf:    make([]complex128, s.dataSize),
		binBuf:     make([]float64, s.maxBins),
		prevBuf:    make([]float64, s.maxBins),
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
	}

	set.fftPlan = fft.New(input, set.dataBuf, s.sampleSize, fft.Estimate)

	s.setCount++

	return set
}

// Recalculate rebuilds our frequency bins with bins bin counts
//
// reference: https://github.com/karlstav/cava/blob/master/cava.c#L654
// reference: https://github.com/noriah/cli-visualizer/blob/master/src/Transformer/SpectrumTransformer.cpp#L598
func (s *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > s.maxBins {
		bins = s.maxBins
	}

	s.numBins = bins

	var cBins = float64(bins + 1)

	var cFreq = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin := 0; xBin <= bins; xBin++ {
		// Fix issue where recalculations may not be accurate due to
		// previous runs
		s.loCuts[xBin] = 0
		s.hiCuts[xBin] = 0

		vFreq := (((float64(xBin+1) / cBins) - 1) * cFreq)
		vFreq = hi * math.Pow(10.0, vFreq)
		vFreq = (vFreq / (s.sampleRate / 2)) * (float64(s.sampleSize) / 4)

		s.loCuts[xBin] = int(math.Floor(vFreq))

		if xBin > 0 {
			if s.loCuts[xBin] <= s.loCuts[xBin-1] {
				s.loCuts[xBin] = s.loCuts[xBin-1] + 1
			}

			s.hiCuts[xBin-1] = s.loCuts[xBin-1]
		}
	}

	return s.numBins
}

// Generate makes numBins and dumps them in the buffer
func (s *Spectrum) Generate(dSet *DataSet) {
	dSet.numBins = s.numBins

	for xBin := 0; xBin <= dSet.numBins; xBin++ {

		var vM = 0.0

		for xF := s.loCuts[xBin]; xF <= s.hiCuts[xBin] &&
			xF >= 0 &&
			xF < dSet.dataSize; xF++ {

			vM += pyt(dSet.dataBuf[xF])
		}

		vM /= float64(s.hiCuts[xBin] - s.loCuts[xBin] + 1)

		vM *= (math.Log2(float64(2+xBin)) * (100.0 / float64(dSet.numBins)))

		dSet.binBuf[xBin] = math.Pow(vM, 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}
