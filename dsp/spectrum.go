package dsp

import (
	"math"

	"github.com/noriah/tavis/fft"
)

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

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBins int
	numBins int

	fftSize int
	fftBuf  []complex128

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	loCuts []int
	hiCuts []int

	eqBins []float64
}

// NewSpectrum will set up our spectrum
func NewSpectrum(hz float64, size, max int) *Spectrum {

	var fftSize = (size / 2) + 1

	var sp = &Spectrum{
		maxBins:    max,
		numBins:    max,
		fftSize:    fftSize,
		fftBuf:     make([]complex128, fftSize),
		sampleSize: size,
		sampleRate: hz,
		loCuts:     make([]int, max+1),
		hiCuts:     make([]int, max+1),
		eqBins:     make([]float64, max+1),
	}

	sp.Recalculate(max, 1, sp.sampleRate/2)

	return sp
}

// BinSet reurns a new data set with settings matching this spectrum
func (sp *Spectrum) BinSet(input []float64) *BinSet {
	return &BinSet{
		buffer: make([]float64, sp.maxBins),
		plan:   fft.NewPlan(input, sp.fftBuf),
	}
}

// Recalculate rebuilds our frequency bins with bins bin counts
func (sp *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > sp.maxBins {
		bins = sp.maxBins
	}

	sp.numBins = bins

	var cBins = float64(bins + 1)

	var cNyquist = (float64(sp.sampleSize) / 4) / (sp.sampleRate / 2)

	var cCoef = 100.0 / float64(bins)

	var cF = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xB := 0; xB <= bins; xB++ {
		var fxB = float64(xB + 1)
		// Fix issue where recalculations may not be accurate due to
		// previous recalculations
		sp.loCuts[xB] = 0
		sp.hiCuts[xB] = 0

		var vFreq = (fxB / (cBins * cF)) - cF
		vFreq = hi * math.Pow(10.0, vFreq) * cNyquist

		sp.loCuts[xB] = int(vFreq)

		if xB > 0 {
			if sp.loCuts[xB] <= sp.loCuts[xB-1] {
				sp.loCuts[xB] = sp.loCuts[xB-1] + 1
			}

			// previous high cutoffs are equal to previous low cuttoffs?
			sp.hiCuts[xB-1] = sp.loCuts[xB-1]

			if sp.hiCuts[xB-1] >= sp.fftSize {
				sp.hiCuts[xB-1] = sp.fftSize - 1
			}

			var diff = sp.hiCuts[xB-1] - sp.loCuts[xB-1] + 1

			sp.eqBins[xB-1] = (math.Log2(fxB) * cCoef) / float64(diff)
		}
	}

	return bins
}

// Generate makes numBins and dumps them in the buffer
func (sp *Spectrum) Generate(bs *BinSet) {

	bs.count = sp.numBins

	bs.plan.Execute()

	for xB := 0; xB < sp.numBins; xB++ {
		var mag = 0.0

		for xF := sp.loCuts[xB]; xF <= sp.hiCuts[xB] && xF >= 0; xF++ {
			mag += pyt(sp.fftBuf[xF])
		}

		bs.buffer[xB] = math.Pow(mag*sp.eqBins[xB], 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}
