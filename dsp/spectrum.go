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

	loCuts []int
	widths []int

	eqBins []float64

	fftBuf []complex128

	streams    []*stream
	streamBufs [][]float64
}

// NewSpectrum will set up our spectrum
func NewSpectrum(hz float64, size int) *Spectrum {

	var fftSize = (size / 2) + 1

	var sp = &Spectrum{
		numBins:    size,
		fftSize:    fftSize,
		sampleSize: size,
		sampleRate: hz,
		gamma:      4.0,
		loCuts:     make([]int, size+1),
		widths:     make([]int, size+1),
		eqBins:     make([]float64, size+1),
		fftBuf:     make([]complex128, fftSize),
		streams:    make([]*stream, 0),
		streamBufs: make([][]float64, 0),
	}

	sp.Recalculate(size)
	sp.SetSmoothing(0.725)

	return sp
}

func hamming(buf []float64, size int) {
	var coef = math.Pi * 2 / float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (0.53836 - 0.46164) * math.Cos(coef*float64(n))
	}
}

// Process makes numBins and dumps them in the buffer
func (sp *Spectrum) Process() {

	var sf = math.Pow(sp.smoothFact, (float64(sp.sampleSize) / sp.sampleRate))

	for _, stream := range sp.streams {

		stream.plan.Execute()

		for xB := 0; xB < sp.numBins; xB++ {
			var mag = 0.0
			var xF = sp.loCuts[xB]
			var xW = sp.widths[xB]

			for m := xF + xW; xF < m && xF < sp.fftSize; xF++ {
				var power = cmplx.Abs(sp.fftBuf[xF])
				if mag < power {
					mag = power
				}
			}

			// mag /= float64(xW)
			mag *= sp.eqBins[xB]

			mag = math.Log(mag)
			if mag < 0.0 {
				mag = 0.0
			}

			mag *= 1.0 - sf
			mag += stream.pBuf[xB] * sf

			stream.pBuf[xB] = mag

			stream.buf[xB] = mag
		}
	}
}

// Frequencies [0] -- Bass -- [1] -- Mid -- [2] -- Treble -- [3]
var dividers = []float64{
	0.0,
	20.0,
	150.0,
	3600.0,
	8000.0,
	22050.0,
}

// Recalculate rebuilds our frequency bins
//
// https://stackoverflow.com/questions/3694918/how-to-extract-frequency-associated-with-fft-values-in-python
//  - https://stackoverflow.com/a/27191172
// https://dlbeer.co.nz/articles/fftvis.html
// https://www.cg.tuwien.ac.at/courses/WissArbeiten/WS2010/processing.pdf
func (sp *Spectrum) Recalculate(bins int) int {
	if bins > sp.sampleSize {
		bins = sp.sampleSize
	}

	sp.numBins = bins

	var halfSampleSize = float64(sp.sampleSize) / 2

	// var T = 1 / (sp.sampleRate / sampleSize / 2)

	// var df = 1 / T

	// var dw = (2 * math.Pi) / T

	// var ny = dw * (sampleSize / 2)

	// - RATE: 44100 | SIZE: 1024
	// - MAX: 512
	// - PASS inside the array and where expected
	// panic(int(math.Floor(22050.0 * T)))

	var cCoef = 100.0 / float64(bins+1)

	// var lo = (dividers[0] + 1) * T
	// var hi = dividers[4] * T

	// var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	var fStart = 0

	for xB := 0; xB <= bins; xB++ {
		// Fix issue where recalculations may not be accurate due to
		// previous recalculations
		sp.loCuts[xB] = fStart
		var vFreq = math.Pow(float64(xB+1)/float64(bins), sp.gamma)
		vFreq *= halfSampleSize
		var fEnd = int(math.Round(vFreq))

		if fEnd > sp.fftSize {
			fEnd = sp.fftSize
		}

		var width = fEnd - fStart
		if width < 1 {
			width = 1
		}

		sp.widths[xB] = width

		// var mel = 700.0 * (math.Exp((float64(xB) / 1127.0)) - 1)

		// var vFreq = ((float64(xB+1) / float64(bins+1)) * cF) - cF
		// vFreq = (math.Pow(10.0, vFreq) * hi)

		// sp.loCuts[xB] = fStart

		// if xB > 0 {
		// 	if sp.loCuts[xB-1] >= sp.loCuts[xB] {
		// 		sp.loCuts[xB] = sp.loCuts[xB-1] + 1

		// 		if xB > 1 {
		// 			// sp.loCuts[xB] += (sp.loCuts[xB-1] - sp.loCuts[xB-2]) - 1
		// 		}
		// 	}

		sp.eqBins[xB] = math.Log2(float64(xB+2)) * cCoef
		// }

		fStart = fEnd
	}

	return bins
}

type stream struct {
	input []float64
	buf   []float64
	pBuf  []float64
	plan  *fft.Plan
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
