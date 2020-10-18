package dsp

import (
	"math"

	"github.com/noriah/tavis/fft"
)

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
	}

	sp.loCuts = make([]int, sp.maxBins+1)
	sp.hiCuts = make([]int, sp.maxBins+1)

	sp.Recalculate(sp.maxBins, 20, sp.sampleRate/2)

	return sp
}

// BinSet reurns a new data set with settings matching this spectrum
func (sp *Spectrum) BinSet(input []float64) *BinSet {

	return &BinSet{
		count:  sp.maxBins,
		buffer: make([]float64, sp.maxBins),
		plan:   fft.NewPlan(input, sp.fftBuf, sp.sampleSize),
	}
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

	var cScale = (float64(sp.sampleSize) / 4) / (sp.sampleRate / 2)

	var cF = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin := 0; xBin <= bins; xBin++ {
		// Fix issue where recalculations may not be accurate due to
		// previous recalculations
		sp.loCuts[xBin] = 0
		sp.hiCuts[xBin] = 0

		vFreq := (((float64(xBin+1) / cBins) - 1) * cF)
		vFreq = hi * math.Pow(10.0, vFreq)
		vFreq = vFreq * cScale

		sp.loCuts[xBin] = int(vFreq)

		if xBin > 0 {
			if sp.loCuts[xBin] <= sp.loCuts[xBin-1] {
				sp.loCuts[xBin] = sp.loCuts[xBin-1] + 1
			}

			// previous high cutoffs are equal to previous low cuttoffs?
			sp.hiCuts[xBin-1] = sp.loCuts[xBin-1]
		}
	}

	return sp.numBins
}

// Generate makes numBins and dumps them in the buffer
func (sp *Spectrum) Generate(bs *BinSet) {

	bs.plan.Execute()

	bs.count = sp.numBins

	var cCoef = 100.0 / float64(bs.count)

	for xBin := 0; xBin < bs.count; xBin++ {

		var vM = 0.0
		var xF = sp.loCuts[xBin]

		for xF <= sp.hiCuts[xBin] && xF >= 0 && xF < sp.fftSize {
			vM += pyt(sp.fftBuf[xF])
			xF++
		}

		// divide bin sum by total frequencies included
		vM /= float64(xF - sp.loCuts[xBin] + 1)

		vM *= math.Log2(float64(xBin+2)) * cCoef

		bs.buffer[xBin] = math.Pow(vM, 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}
