package dsp

import (
	"math"
	"math/cmplx"

	"github.com/noriah/tavis/fft"
)

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	numBins    int
	numStreams int

	fftSize int

	sampleSize int
	sampleRate float64

	gamma float64

	smoothFact float64
	smoothResp float64

	bins []bin

	fftBuf []complex128

	streams    []*stream
	streamBufs [][]float64
}

type bin struct {
	eqVal float64

	floorFFT int
	ceilFFT  int
	widthFFT int
}

type stream struct {
	input []float64
	buf   []float64
	pBuf  []float64
	plan  *fft.Plan
}

// Some notes:
//
// https://stackoverflow.com/questions/3694918/how-to-extract-frequency-associated-with-fft-values-in-python
//  - https://stackoverflow.com/a/27191172
// https://dlbeer.co.nz/articles/fftvis.html
// https://github.com/hvianna/audioMotion-analyzer/blob/master/src/audioMotion-analyzer.js#L1053
// https://www.cg.tuwien.ac.at/courses/WissArbeiten/WS2010/processing.pdf
// https://dsp.stackexchange.com/questions/6499/help-calculating-understanding-the-mfccs-mel-frequency-cepstrum-coefficients

// NewSpectrum will set up our spectrum
func NewSpectrum(hz float64, size int) *Spectrum {

	var fftSize = (size / 2) + 1

	var sp = &Spectrum{
		numBins:    size,
		fftSize:    fftSize,
		sampleSize: size,
		sampleRate: hz,
		smoothFact: 0.255,
		gamma:      4.0,
		bins:       make([]bin, size+1),
		fftBuf:     make([]complex128, fftSize),
		streams:    make([]*stream, 0),
		streamBufs: make([][]float64, 0),
	}

	return sp
}

// Process makes numBins and dumps them in the buffer
func (sp *Spectrum) Process() {
	var sf = math.Pow(10.0, (-(1 - sp.smoothFact))*10.0)

	sf = math.Pow(sf, float64(sp.sampleSize)/sp.sampleRate)

	var pct = 0.001 * (float64(sp.sampleSize) * 0.5)

	for _, stream := range sp.streams {

		// window.Hamming(stream.input, sp.sampleSize)

		stream.plan.Execute()

		for xB := 0; xB < sp.numBins; xB++ {
			var mag = 0.0

			var xF = sp.bins[xB].floorFFT
			for xF < sp.bins[xB].ceilFFT && xF < sp.fftSize {
				// if power := cmplx.Abs(sp.fftBuf[xF]); mag < power {
				// 	mag = power
				// }
				mag += cmplx.Abs(sp.fftBuf[xF])
				xF++
			}

			mag /= float64(sp.bins[xB].widthFFT)
			mag *= sp.bins[xB].eqVal

			switch {
			case xF < int(pct):
				mag = math.Pow(mag, 0.5*(float64(xF)/pct))
			default:
				mag = math.Pow(mag, 0.55)
			}

			if mag < 0.0 {
				mag = 0.0
			}

			// Smoothing

			// mag *= (1.0 - sf)
			// mag += stream.pBuf[xB] * sf
			// stream.pBuf[xB] = mag
			// stream.buf[xB] = mag

			mag += stream.pBuf[xB] * sp.smoothFact
			stream.pBuf[xB] = mag * (1 - (1 / (1 + (mag * 1))))
			stream.buf[xB] = mag

		}
	}
}

// Frequencies
// [0] - Sub - [1] Bass - [2] Mid - [3] - Treble - [4] - Brilliance - [5]
var dividers = []float64{
	20.0,
	60.0,
	250.0,
	4000.0,
	12000.0,
	22050.0,
}

// Recalculate rebuilds our frequency bins
func (sp *Spectrum) Recalculate(bins int) int {
	if bins > sp.fftSize {
		bins = sp.fftSize
	}

	sp.numBins = bins

	// var cCoef = 1.0 / float64(bins+1)

	// clean the bins
	for xB := 0; xB < bins; xB++ {
		sp.bins[xB] = bin{
			eqVal: math.Log10(float64(xB) + 2),
		}
	}

	var loF = dividers[0]
	var hiF = math.Min(dividers[4], sp.sampleRate/2)
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

	for xB := 0; xB < bins; xB++ {
		var b = sp.bins[xB]

		if b.ceilFFT == b.floorFFT {
			sp.bins[xB].widthFFT = 1
			continue
		}

		sp.bins[xB].widthFFT = b.ceilFFT - b.floorFFT
	}

	return bins
}

func (sp *Spectrum) idxToFreq(bin int) float64 {
	return float64(bin) * sp.sampleRate / float64(sp.sampleSize)
}

func (sp *Spectrum) freqToIdx(freq float64, round func(float64) float64) int {
	var bin = int(round(freq / (sp.sampleRate / float64(sp.sampleSize))))

	if bin < sp.fftSize {
		return bin
	}

	return sp.fftSize - 1
}

// AddStream adds an input buffer to the spectrum
func (sp *Spectrum) AddStream(input []float64) bool {
	var s = &stream{
		input: input,
		buf:   make([]float64, sp.sampleSize),
		pBuf:  make([]float64, sp.sampleSize),
		plan:  fft.NewPlan(input, sp.fftBuf),
	}

	sp.streamBufs = append(sp.streamBufs, s.buf)
	sp.streams = append(sp.streams, s)
	sp.numStreams++

	return true
}

// SetGamma sets the gamma used for distribution spread
func (sp *Spectrum) SetGamma(g float64) {
	if g <= 0.0 {
		g = 1
	}

	sp.gamma = g
}

// SetSmoothing sets the smoothing parameters
func (sp *Spectrum) SetSmoothing(factor float64) {
	if factor <= 0 {
		factor = math.SmallestNonzeroFloat64
	}

	sp.smoothFact = factor
}

// Buffers returns our bin buffers
func (sp *Spectrum) Buffers() [][]float64 {
	return sp.streamBufs
}

// StreamCount returns the number of streams in our buffers
func (sp *Spectrum) StreamCount() int {
	return sp.numStreams
}

// BinCount returns the number of bins each stream has
func (sp *Spectrum) BinCount() int {
	return sp.numBins
}
