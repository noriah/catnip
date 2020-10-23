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

	ceilFFT  int
	floorFFT int
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
		bins:       make([]bin, size+1),
		fftBuf:     make([]complex128, fftSize),
		streams:    make([]*stream, 0),
		streamBufs: make([][]float64, 0),
	}

	sp.Recalculate(size)
	sp.SetSmoothing(0.725)

	return sp
}

func hamming(buf []float64, size int) {
	var coef = 2 * math.Pi / float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (0.53836 - 0.46164*math.Cos(coef*float64(n)))
	}
}

func hann(buf []float64, size int) {
	var coef = 2 * math.Pi / float64(size)
	for n := 0; n < size; n++ {
		buf[n] *= (0.5 - 0.5*math.Cos(coef*float64(n)))
	}
}

// Process makes numBins and dumps them in the buffer
func (sp *Spectrum) Process() {

	// var sf = math.Pow(sp.smoothFact/10000, (float64(sp.sampleSize) / sp.sampleRate))

	for _, stream := range sp.streams {

		// hamming(stream.input, sp.sampleSize)

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

			mag /= float64(sp.bins[xB].ceilFFT - sp.bins[xB].floorFFT)
			mag *= sp.bins[xB].eqVal
			mag = math.Pow(mag, 0.65)
			// mag = math.Pow(mag, 2)

			if mag < 0.0 {
				mag = 0.0
			}

			// mag *= (1.0 - sf)
			// mag += stream.pBuf[xB] * sf

			mag += stream.pBuf[xB] * sp.smoothFact

			stream.pBuf[xB] = mag * (1 - (1 / (1 + (mag * 10))))
			// stream.pBuf[xB] = mag

			stream.buf[xB] = mag

		}

		Monstercat(stream.buf, sp.numBins, 1.75)
	}
}

// Frequencies [0] -- Bass -- [1] -- Mid -- [2] -- Treble -- [3]
var dividers = []float64{
	0.0,
	20.0,
	150.0,
	3600.0,
	6000.0,
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

	var cCoef = 100.0 / float64(bins+1)

	// clean the bins
	for xB := 0; xB < bins; xB++ {
		sp.bins[xB] = bin{
			floorFFT: sp.fftSize,

			eqVal: math.Log2(float64(xB+2)) * cCoef,
		}
	}

	// if false {
	// 	for f := 1; f < sp.fftSize-1; f++ {
	// 		var xB = int(
	// 			math.Floor(
	// 				math.Pow((float64(f)/float64(sp.fftSize)), 1/sp.gamma) * float64(bins)),
	// 		)
	// 		if sp.bins[xB].ceilFFT < f {
	// 			sp.bins[xB].ceilFFT = f + 1
	// 		}

	// 		if sp.bins[xB].floorFFT > f {
	// 			sp.bins[xB].floorFFT = f
	// 		}
	// 	}
	// }

	// if true {

	var T = 1 / (sp.sampleRate / float64(sp.sampleSize))

	var lo = (dividers[0] + 1)
	var hi = dividers[4]

	var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	for xB := 0; xB <= bins; xB++ {
		var vFreq = ((float64(xB+1) / float64(bins+1)) * cF) - cF
		vFreq = (math.Pow(10.0, vFreq) * hi) * T

		sp.bins[xB].floorFFT = int(math.Floor(vFreq))

		if xB > 0 {
			if sp.bins[xB-1].floorFFT >= sp.bins[xB].floorFFT {
				sp.bins[xB].floorFFT = sp.bins[xB-1].floorFFT + 1

				// if xB > 1 {
				// 	sp.bins[xB].floorFFT += sp.bins[xB-1].floorFFT
				// 	sp.bins[xB].floorFFT -= sp.bins[xB-2].floorFFT + 1
				// }
			}

			sp.bins[xB-1].ceilFFT = sp.bins[xB-1].floorFFT
		}
	}
	// }

	// if false {

	// 	var T = 1 / (sp.sampleRate / float64(sp.sampleSize) / 2)

	// 	// var lo = (dividers[1] + 1)
	// 	// var hi = dividers[4]

	// 	var fss = math.Min(dividers[4], sp.sampleRate/2) * T

	// 	// var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	// 	for xB := 0; xB <= bins; xB++ {
	// 		// var vFreq = ((float64(xB+1) / float64(bins+1)) * cF) - cF
	// 		var vFreq = float64(xB-bins) / (float64(bins))
	// 		vFreq = (math.Pow(10.0, vFreq) * fss)

	// 		sp.bins[xB].floorFFT = int(math.Floor(vFreq))

	// 		if xB > 0 {
	// 			if sp.bins[xB].floorFFT <= sp.bins[xB-1].floorFFT {
	// 				sp.bins[xB].floorFFT = sp.bins[xB-1].floorFFT + 1

	// 				// if xB > 1 {
	// 				// 	sp.bins[xB].floorFFT += sp.bins[xB-1].floorFFT
	// 				// 	sp.bins[xB].floorFFT -= sp.bins[xB-2].floorFFT + 1
	// 				// }
	// 			}

	// 			sp.bins[xB-1].ceilFFT = sp.bins[xB].floorFFT

	// 		}
	// 	}
	// }

	// for xB := 0; xB < bins; xB++ {
	// 	var b = sp.bins[xB]
	// 	fmt.Println(xB, b.floorFFT, b.ceilFFT)
	// }

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
