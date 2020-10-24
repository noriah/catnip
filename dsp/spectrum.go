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
		gamma:      4.0,
		bins:       make([]bin, size+1),
		fftBuf:     make([]complex128, fftSize),
		streams:    make([]*stream, 0),
		streamBufs: make([][]float64, 0),
	}

	// sp.Recalculate(size)
	sp.SetSmoothing(0.725)

	return sp
}

// window function
// https://www.wikiwand.com/en/Window_function

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

	// var sf = math.Pow(math.Pow(sp.smoothFact, 50), (float64(sp.sampleSize) / sp.sampleRate))

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

			mag /= float64(xF - sp.bins[xB].floorFFT)

			mag *= sp.bins[xB].eqVal
			mag = math.Pow(mag, 0.65)
			// mag = math.Pow(mag, 2)

			if mag < 0.0 {
				mag = 0.0
			}

			// Smoothing

			// mag *= (1.0 - sf)
			// mag += stream.pBuf[xB] * sf
			// stream.pBuf[xB] = mag
			stream.buf[xB] = mag

			mag += stream.pBuf[xB] * sp.smoothFact
			stream.pBuf[xB] = mag * (1 - (1 / (1 + (mag * 1))))
			stream.buf[xB] = mag

		}

		// Monstercat(stream.buf, sp.numBins, 1.99)
	}
}

// Frequencies [0] -- Bass -- [1] -- Mid -- [2] -- Treble -- [3]
var dividers = []float64{
	0.0,
	60.0,
	150.0,
	4000.0,
	12000.0,
	22050.0,
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

// Recalculate rebuilds our frequency bins
func (sp *Spectrum) Recalculate(bins int) int {
	if bins > sp.fftSize {
		bins = sp.fftSize
	}

	sp.numBins = bins

	var cCoef = 100.0 / float64(bins+1)

	// clean the bins
	for xB := 0; xB < bins; xB++ {
		sp.bins[xB] = bin{
			eqVal: math.Log2(float64(xB+2)) * cCoef,
		}
	}

	// if false {
	// 	var loF = dividers[1]
	// 	var hiF = math.Min(dividers[4], sp.sampleRate/2)
	// 	var minIdx = sp.freqToIdx(loF, math.Floor)
	// 	var maxIdx = sp.freqToIdx(hiF, math.Round)

	// 	var size = maxIdx - minIdx

	// 	var spread = size / bins
	// 	if spread < 1 {
	// 		spread++
	// 	}
	// 	var last = 0
	// 	if size%spread > 0 {
	// 		last = sp.fftSize % spread
	// 	}

	// 	var start = minIdx
	// 	var b = bins
	// 	if last > 0 {
	// 		b--
	// 	}
	// 	for xB := 0; xB < b; xB++ {
	// 		sp.bins[xB].widthFFT = spread
	// 		sp.bins[xB].floorFFT = start
	// 		start += spread

	// 		sp.bins[xB].ceilFFT = start
	// 	}

	// 	if last > 0 {
	// 		sp.bins[b].floorFFT = start
	// 		sp.bins[b].ceilFFT = start + last
	// 		sp.bins[b].widthFFT = last
	// 	}
	// }

	// another  one

	// if false {

	// 	var loF = dividers[2]
	// 	var hiF = math.Min(dividers[4], sp.sampleRate/2)

	// 	var minLog = math.Log10(loF)
	// 	var maxLog = math.Log10(hiF)
	// 	var bandWidth = (maxLog - minLog) / float64(bins)

	// 	// var minIdx = sp.freqToIdx(loF, math.Floor)
	// 	// var maxIdx = sp.freqToIdx(hiF, math.Round)

	// 	for xB := 0; xB <= bins; xB++ {
	// 		var fbL = sp.freqToIdx(math.Pow(10.0, (float64(xB)*bandWidth)+minLog), math.Floor)
	// 		sp.bins[xB].floorFFT = fbL

	// 		if xB > 0 {
	// 			var lfbL = sp.bins[xB-1].floorFFT
	// 			if lfbL == fbL {
	// 				fbL++
	// 			}
	// 			sp.bins[xB-1].ceilFFT = fbL
	// 			sp.bins[xB-1].widthFFT = fbL - lfbL
	// 		}
	// 	}
	// }

	// if false {
	// 	for f := 0; f < sp.fftSize; f++ {
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

	// another attempt

	// if true {

	var lo = (dividers[0] + 1)
	var hi = dividers[4]

	var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	hi *= 1 / (sp.sampleRate / float64(sp.sampleSize))

	for xB := 0; xB <= bins; xB++ {
		var vFreq = ((float64(xB) / float64(bins)) * cF) - cF
		vFreq = (math.Pow(10.0, vFreq) * hi)

		sp.bins[xB].floorFFT = int(math.Floor(vFreq))

		if xB > 0 {
			if sp.bins[xB-1].floorFFT >= sp.bins[xB].floorFFT {
				sp.bins[xB].floorFFT = sp.bins[xB-1].floorFFT + 1

				if xB > 1 {
					sp.bins[xB].floorFFT += sp.bins[xB-1].floorFFT
					sp.bins[xB].floorFFT -= sp.bins[xB-2].floorFFT + 1
				}
			}

			sp.bins[xB-1].ceilFFT = sp.bins[xB].floorFFT
			sp.bins[xB-1].widthFFT = sp.bins[xB-1].ceilFFT - sp.bins[xB-1].floorFFT
		}
	}
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

// var fBins = float64(bins)

// var bassRange = math.Log10(dividers[2]) - math.Log10(dividers[1])
// var midRange = math.Log10(dividers[3]) - math.Log10(dividers[2])
// var trebRange = math.Log10(dividers[4]) - math.Log10(dividers[3])

// var total = bassRange + midRange + trebRange
// var bassRatio = bassRange / total
// var midRatio = midRange / total
// var trebRatio = trebRange / total
// var bassBins = fBins * bassRatio
// var midBins = fBins * midRatio
// var trebBins = fBins * trebRatio

// fmt.Println(bassRange, midRange, trebRange)
// fmt.Println(bassRatio, midRatio, trebRatio)
// fmt.Println(bassBins, midBins, trebBins)

// var bassRange = sp.freqToIdx(dividers[2], math.Round) - sp.freqToIdx(dividers[1], math.Floor)
// var midRange = sp.freqToIdx(dividers[3], math.Round) - sp.freqToIdx(dividers[2], math.Floor)
// var trebRange = sp.freqToIdx(dividers[4], math.Round) - sp.freqToIdx(dividers[3], math.Floor)

// var total = float64(bassRange + midRange + trebRange)
// var bassRatio = float64(bassRange) / total
// var midRatio = float64(midRange) / total
// var trebRatio = float64(trebRange) / total
// var bassBins = fBins * bassRatio
// var midBins = fBins * midRatio
// var trebBins = fBins * trebRatio

// fmt.Println(bassRange, midRange, trebRange)
// fmt.Println(bassRatio, midRatio, trebRatio)
// fmt.Println(bassBins, midBins, trebBins)

// Monstercat does monstercat "smoothing"
//
// https://github.com/karlstav/cava/blob/master/cava.c#L157
//
// TODO(winter): make faster (rewrite)
//	slow and hungry as heck!
//	lets look into SIMD
func Monstercat(bins []float64, count int, factor float64) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var vFactP = math.Log(factor)

	for xBin := 1; xBin < count; xBin++ {

		for xTrgt := 0; xTrgt < count; xTrgt++ {

			if xBin != xTrgt {
				var tmp = bins[xBin]
				tmp /= math.Exp(vFactP * math.Abs(float64(xBin-xTrgt)))

				if tmp > bins[xTrgt] {
					bins[xTrgt] = tmp
				}
			}
		}
	}
}
