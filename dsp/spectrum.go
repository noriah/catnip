package dsp

import (
	"math"
	"math/cmplx"

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
func NewSpectrum(hz float64, size int) *Spectrum {

	var fftSize = (size / 2) + 1

	var sp = &Spectrum{
		numBins:    size,
		fftSize:    fftSize,
		fftBuf:     make([]complex128, fftSize),
		sampleSize: size,
		sampleRate: hz,
		loCuts:     make([]int, size+1),
		hiCuts:     make([]int, size+1),
		eqBins:     make([]float64, size+1),
	}

	sp.Recalculate(size)

	return sp
}

// BinSet reurns a new data set with settings matching this spectrum
func (sp *Spectrum) BinSet(input []float64) *BinSet {
	return &BinSet{
		buffer: make([]float64, sp.sampleSize),
		plan:   fft.NewPlan(input, sp.fftBuf),
	}
}

// Generate makes numBins and dumps them in the buffer
func (sp *Spectrum) Generate(bs *BinSet) {

	bs.count = sp.numBins

	bs.plan.Execute()

	for xB := 0; xB < sp.numBins; xB++ {
		var mag = 0.0

		for xF := sp.loCuts[xB]; xF < sp.hiCuts[xB] && xF >= 0; xF++ {
			mag += cmplx.Abs(sp.fftBuf[xF])
		}

		bs.buffer[xB] = math.Pow(mag*sp.eqBins[xB], 0.5)
		// bs.buffer[xB] = math.Pow(mag, 0.5)
	}
}

// Frequencies [0] -- Bass -- [1] -- Mid -- [2] -- Treble -- [3]
var dividers = []float64{
	20.0,
	150.0,
	3600.0,
	16000.0,
}

// Recalculate rebuilds our frequency bins
//
// https://stackoverflow.com/questions/3694918/how-to-extract-frequency-associated-with-fft-values-in-python
//  - https://stackoverflow.com/a/27191172
// https://www.cg.tuwien.ac.at/courses/WissArbeiten/WS2010/processing.pdf
func (sp *Spectrum) Recalculate(bins int) int {
	if bins > sp.sampleSize {
		bins = sp.sampleSize
	}

	sp.numBins = bins

	// var dt = 1 / (sp.sampleRate / float64(sp.sampleSize))

	// var T = dt * float64(sp.fftSize)

	// var df = 1 / T

	// var dw = 2 * math.Pi / T

	// var ny = dw * sampleSize / 2

	// - RATE: 44100 | SIZE: 1024
	// - MAX: 512
	// - PASS inside the array and where expected
	// var max = int(math.Floor(22050.0 * T))
	// panic(max)

	// var cCoef = 100.0 / float64(bins)

	var lo = dividers[0]
	var hi = dividers[3]

	var cF = math.Log10(lo/hi) / (1/float64(bins+1) - 1)

	for xB := 0; xB <= bins; xB++ {
		// Fix issue where recalculations may not be accurate due to
		// previous recalculations
		sp.loCuts[xB] = 0
		sp.hiCuts[xB] = 0

		var vFreq = (((float64(xB+1) / float64(bins+1)) * cF) - cF)
		vFreq = (math.Pow(10.0, vFreq) * hi) / (sp.sampleRate / 2)
		vFreq *= float64(sp.sampleSize / 4)

		sp.loCuts[xB] = int(math.Floor(vFreq))

		if xB > 0 {
			if sp.loCuts[xB] <= sp.loCuts[xB-1] {
				sp.loCuts[xB] = sp.loCuts[xB-1] + 1
			}

			sp.hiCuts[xB-1] = sp.loCuts[xB]

			if sp.hiCuts[xB-1] >= sp.fftSize {
				sp.hiCuts[xB-1] = sp.fftSize - 1
			}

			var diff = sp.hiCuts[xB-1] - sp.loCuts[xB-1] + 1

			sp.eqBins[xB-1] = math.Log2(float64(xB+2)) / float64(diff)
		}
	}

	return bins
}
