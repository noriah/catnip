// Package dsp provides audio analysis
//
// Some notes:
//
// https://dlbeer.co.nz/articles/fftvis.html
// https://www.cg.tuwien.ac.at/courses/WissArbeiten/WS2010/processing.pdf
// https://github.com/hvianna/audioMotion-analyzer/blob/master/src/audioMotion-analyzer.js#L1053
// https://dsp.stackexchange.com/questions/6499/help-calculating-understanding-the-mfccs-mel-frequency-cepstrum-coefficients
// https://stackoverflow.com/questions/3694918/how-to-extract-frequency-associated-with-fft-values-in-python
//  - https://stackoverflow.com/a/27191172
//
package dsp

import (
	"math"
	"math/cmplx"

	"github.com/noriah/tavis/dsp/window"
	"github.com/noriah/tavis/fft"
)

// SpectrumType is the type of calculation we run
type SpectrumType int

// Spectrum calculation types
const (
	SpectrumLog SpectrumType = iota
	SpectrumEqual

	// SpectrumDefault is the default spectrum type
	SpectrumDefault = SpectrumLog
)

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	numBins      int          // number of bins we look at
	fftSize      int          // number of fft bins
	sampleSize   int          // number of samples per slice
	sType        SpectrumType // the type of spectrum distribution
	sampleRate   float64      // audio sample rate
	winVar       float64      // window variable
	smoothFactor float64      // smothing factor
	fftBuf       []complex128 // fft return buffer
	bins         []bin        // bins for processing
	streams      []*stream    // streams of data
}

type bin struct {
	eqVal    float64 // equalizer value
	floorFFT int     // floor fft index
	ceilFFT  int     // ceiling fft index
	widthFFT int     // fft floor-ceiling index delta
}

type stream struct {
	input []float64 // input data buffer
	buf   []float64 // bar bin buffer
	pBuf  []float64 // previous run bin buffer
	plan  *fft.Plan // fft plan
}

// Frequencies are the dividing frequencies
var Frequencies = []float64{
	// sub sub bass
	20.0, // 0
	// sub bass
	60.0, // 1
	// bass
	250.0, // 2
	// midrange
	4000.0, // 3
	// treble
	12000.0, // 4
	// brilliance
	22050.0, // 5
	// everything else
}

// NewSpectrum will set up our spectrum
func NewSpectrum(hz float64, size int) *Spectrum {

	var fftSize = (size / 2) + 1

	var sp = &Spectrum{
		fftSize:      fftSize,
		sampleSize:   size,
		sampleRate:   hz,
		smoothFactor: 0.1969,
		winVar:       1.0,
		fftBuf:       make([]complex128, fftSize),
		bins:         make([]bin, size+1),
		streams:      make([]*stream, 0, 2),
	}

	return sp
}

// StreamCount returns the number of streams in our buffers
func (sp *Spectrum) StreamCount() int {
	return len(sp.streams)
}

// AddStream adds an input buffer to the spectrum
func (sp *Spectrum) AddStream(input []float64) {
	var s = &stream{
		input: input,
		buf:   make([]float64, sp.sampleSize),
		pBuf:  make([]float64, sp.sampleSize),
		plan:  fft.NewPlan(input, sp.fftBuf),
	}

	sp.streams = append(sp.streams, s)
}

// BinBuffers returns our bin buffers
func (sp *Spectrum) BinBuffers() [][]float64 {
	var buf = make([][]float64, len(sp.streams))
	for idx, stream := range sp.streams {
		buf[idx] = stream.buf
	}

	return buf
}

// BinCount returns the number of bins each stream has
func (sp *Spectrum) BinCount() int {
	return sp.numBins
}

// Process makes numBins and dumps them in the buffer
func (sp *Spectrum) Process(win window.Function) {
	var sf = math.Pow(10.0, (1-sp.smoothFactor)*(-10.0))

	sf = math.Pow(sf, float64(sp.sampleSize)/sp.sampleRate)

	var bassCut = sp.freqToIdx(Frequencies[2], math.Round)
	var fBassCut = float64(bassCut)

	for _, stream := range sp.streams {

		win(stream.input)

		stream.plan.Execute()

		for xB := 0; xB < sp.numBins; xB++ {
			var mag = 0.0

			var xF = sp.bins[xB].floorFFT
			for xF < sp.bins[xB].ceilFFT && xF < sp.fftSize {
				if power := cmplx.Abs(sp.fftBuf[xF]); mag < power {
					mag = power
				}
				// mag += cmplx.Abs(sp.fftBuf[xF])
				xF++
			}

			// mag /= float64(sp.bins[xB].widthFFT)

			var pow = 0.65

			switch {
			case mag < 0.0:
				mag = 0.0
				continue

			case sp.bins[xB].floorFFT < bassCut:
				pow *= math.Max(0.6, float64(xF)/fBassCut)

			}

			mag *= sp.bins[xB].eqVal

			mag = math.Pow(mag, pow)

			// time smoothing

			mag *= (1.0 - sf)
			mag += stream.pBuf[xB] * sf
			stream.pBuf[xB] = mag
			stream.buf[xB] = mag

			// mag += stream.pBuf[xB] * sp.smoothFactor
			// stream.pBuf[xB] = mag * (1 - (1 / (1 + (mag * 2))))
			// stream.buf[xB] = mag

		}
	}
}

// Recalculate rebuilds our frequency bins
func (sp *Spectrum) Recalculate(bins int) int {

	switch {
	case bins >= sp.fftSize:
		bins = sp.fftSize - 1
	case bins == sp.numBins:
		return bins
	}

	sp.numBins = bins

	// clean the bins
	for xB := 0; xB < bins; xB++ {
		sp.bins[xB].floorFFT = 0
		sp.bins[xB].ceilFFT = 0

		sp.bins[xB].eqVal = 1.0
	}

	switch sp.sType {

	case SpectrumLog:
		sp.distributeLog(bins)

	case SpectrumEqual:
		sp.distributeEqual(bins)

	default:
	}

	// set widths
	for xB := 0; xB < bins; xB++ {
		if sp.bins[xB].ceilFFT == sp.bins[xB].floorFFT {
			sp.bins[xB].widthFFT = 1
			continue
		}

		sp.bins[xB].widthFFT = sp.bins[xB].ceilFFT - sp.bins[xB].floorFFT
	}

	return bins
}

// distributeLog does not *actually* distribute logarithmically
// it is a best guess naive attempt right now.
// i will continue work on it - winter
func (sp *Spectrum) distributeLog(bins int) {
	var lo = (Frequencies[1])
	var hi = Frequencies[4]

	var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	var getBinBase = func(b int) int {
		var vFreq = ((float64(b+1) / float64(bins)) * cF) - cF
		vFreq = math.Pow(10.0, vFreq) * hi
		return sp.freqToIdx(vFreq, math.Round)
	}

	var cCoef = 100.0 / float64(bins+1)

	sp.bins[0].floorFFT = lo &

	for xB := 0; xB < bins; xB++ {

		sp.bins[xB].floorFFT = getBinBase(xB)
		sp.bins[xB].eqVal = math.Log2(float64(xB)+2) * cCoef

		if xB > 0 {
			if sp.bins[xB-1].floorFFT >= sp.bins[xB].floorFFT {
				sp.bins[xB].floorFFT = sp.bins[xB-1].floorFFT + 1
			}

			sp.bins[xB-1].ceilFFT = sp.bins[xB].floorFFT
		}
	}
}

func (sp *Spectrum) distributeEqual(bins int) {
	var loF = Frequencies[0]
	var hiF = math.Min(Frequencies[4], sp.sampleRate/2)
	var minIdx = sp.freqToIdx(loF, math.Floor)
	var maxIdx = sp.freqToIdx(hiF, math.Round)

	var size = maxIdx - minIdx

	var spread = size / bins

	if spread < 1 {
		spread++
	}

	var last = size % spread

	var start = minIdx
	var lBins = bins
	if last > 0 {
		lBins--
	}

	for xB := 0; xB < lBins; xB++ {
		sp.bins[xB].widthFFT = spread
		sp.bins[xB].floorFFT = start
		start += spread

		sp.bins[xB].ceilFFT = start
	}

	if last > 0 {
		sp.bins[lBins].floorFFT = start
		sp.bins[lBins].ceilFFT = start + last
		sp.bins[lBins].widthFFT = last
	}
}

func (sp *Spectrum) idxToFreq(bin int) float64 {
	return float64(bin) * sp.sampleRate / float64(sp.sampleSize)
}

type mathFunc func(float64) float64

func (sp *Spectrum) freqToIdx(freq float64, round mathFunc) int {
	var bin = int(round(freq / (sp.sampleRate / float64(sp.sampleSize))))

	if bin < sp.fftSize {
		return bin
	}

	return sp.fftSize - 1
}

// SetType will set the spectrum type
func (sp *Spectrum) SetType(st SpectrumType) {
	sp.sType = st
}

// SetWinVar sets the winVar used for distribution spread
func (sp *Spectrum) SetWinVar(g float64) {
	switch {

	case g <= 0.0:
		sp.winVar = 1.0

	default:
		sp.winVar = g

	}

}

// SetSmoothing sets the smoothing parameters
func (sp *Spectrum) SetSmoothing(factor float64) {
	switch {

	case factor <= 0.0:
		sp.smoothFactor = math.SmallestNonzeroFloat64

	default:
		sp.smoothFactor = factor

	}

}
