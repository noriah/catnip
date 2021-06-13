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
)

// SpectrumType is the type of distribution we use
type SpectrumType int

// Spectrum distribution types
const (
	TypeLog SpectrumType = iota
	TypeEqual
	TypeLog2

	// SpectrumDefault is the default spectrum distribution
	TypeDefault = TypeLog
)

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	Bins         BinBuf       // bins for processing
	SampleSize   int          // number of samples per slice
	numBins      int          // number of bins we look at
	fftSize      int          // number of fft bins
	sType        SpectrumType // the type of spectrum distribution
	SampleRate   float64      // audio sample rate
	winVar       float64      // window variable
	smoothFactor float64      // smothing factor
	smoothPow    float64      // smoothing pow
}

// Bin is a helper struct for spectrum
type Bin struct {
	powVal   float64 // powpow
	eqVal    float64 // equalizer value
	floorFFT int     // floor fft index
	ceilFFT  int     // ceiling fft index
	// widthFFT int     // fft floor-ceiling index delta
}

// BinBuf is an alias for a slice of Bins
type BinBuf = []Bin

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
	8000.0, // 4
	// brilliance
	22050.0, // 5
	// everything else
}

// BinCount returns the number of bins each stream has
func (sp *Spectrum) BinCount() int {
	return sp.numBins
}

// Process makes numBins and dumps them in the buffer
func (sp *Spectrum) Process(dest []float64, src []complex128) {
	for xB, bin := range sp.Bins[:sp.numBins] {
		dest[xB] *= sp.smoothPow

		var mag = 0.0

		for xF := bin.floorFFT; xF < bin.ceilFFT && xF < sp.fftSize; xF++ {
			if power := math.Hypot(real(src[xF]), imag(src[xF])); mag < power {
				mag = power
			}
		}

		if mag <= 0.0 {
			continue
		}

		// time smoothing
		dest[xB] += math.Pow(mag, bin.powVal) * (1.0 - sp.smoothPow)
	}
}

func (sp *Spectrum) ProcessBin(idx int, scale, old float64, src []complex128) float64 {
	mag := 0.0
	bin := sp.Bins[idx]

	fftFloor, fftCeil := bin.floorFFT, bin.ceilFFT
	if fftCeil > sp.fftSize {
		fftCeil = sp.fftSize
	}

	src = src[fftFloor:fftCeil]
	for _, cmplx := range src {
		power := math.Hypot(real(cmplx), imag(cmplx))
		if mag < power {
			mag = power
		}
	}

	// mag /= float64(fftCeil - fftFloor)

	// if mag <= 0.0 {
	// 	return old * sp.smoothPow
	// }

	// time smoothing
	mag = math.Pow(mag, bin.powVal) * (1.0 - sp.smoothPow)

	// reduce mag by an amount to remove noise.
	// this could change over time with song.
	// maybe look into a moving window of some value.
	mag = math.Max(mag-(0.015*scale), 0.0)

	return (old * sp.smoothPow) + mag
}

// Recalculate rebuilds our frequency bins
func (sp *Spectrum) Recalculate(bins int) int {
	if sp.fftSize == 0 {
		sp.fftSize = sp.SampleSize/2 + 1
	}

	switch {
	case bins >= sp.fftSize:
		bins = sp.fftSize - 1
	case bins == sp.numBins:
		return bins
	}

	sp.numBins = bins

	// clean the bins
	for idx := range sp.Bins[:bins] {
		sp.Bins[idx] = Bin{
			powVal: 0.65,
			eqVal:  1.0,
		}
	}

	switch sp.sType {

	case TypeLog:
		sp.distributeLog(bins)

	case TypeEqual:
		sp.distributeEqual(bins)

	case TypeLog2:
		sp.distributeLog2(bins)

	default:
		return bins

	}

	var bassCut = sp.freqToIdx(Frequencies[2], math.Floor)
	var fBassCut = float64(bassCut)

	// set widths
	for idx, b := range sp.Bins[:bins] {
		if b.ceilFFT >= sp.fftSize {
			sp.Bins[idx].ceilFFT = sp.fftSize - 1
		}

		// sp.Bins[idx].widthFFT = b.ceilFFT - b.floorFFT

		if b.ceilFFT <= bassCut {
			sp.Bins[idx].powVal *= math.Max(0.5, float64(b.ceilFFT)/fBassCut)
		}

	}

	return bins
}

// distributeLog *does not actually distribute logarithmically*
// it is a best guess naive attempt right now.
// i will continue work on it - winter
func (sp *Spectrum) distributeLog(bins int) {
	var lo = Frequencies[1]
	var hi = math.Min(sp.SampleRate/2, Frequencies[4])

	var loLog = math.Log10(lo)
	var hiLog = math.Log10(hi)

	var cF = (hiLog - loLog) / float64(bins)

	var cCoef = 100.0 / float64(bins+1)

	for idx := range sp.Bins[:bins+1] {

		var vFreq = ((float64(idx) * cF) + loLog)
		vFreq = math.Pow(10.0, vFreq)
		fftIdx := sp.freqToIdx(vFreq, math.Floor)
		sp.Bins[idx].floorFFT = fftIdx
		sp.Bins[idx].eqVal = math.Log2(float64(fftIdx)+2) * cCoef

		if idx > 0 {
			if sp.Bins[idx-1].floorFFT >= sp.Bins[idx].floorFFT {
				sp.Bins[idx].floorFFT = sp.Bins[idx-1].floorFFT + 1
			}

			sp.Bins[idx-1].ceilFFT = sp.Bins[idx].floorFFT
		}
	}
}

// distributeLog2 does not *actually* distribute logarithmically
// it is a best guess naive attempt right now.
// i will continue work on it - winter
func (sp *Spectrum) distributeLog2(bins int) {
	var lo = Frequencies[1]
	var hi = math.Min(sp.SampleRate/2, Frequencies[4])

	var cF = math.Log10(lo/hi) / ((1 / float64(bins)) - 1)

	var cCoef = 100.0 / float64(bins+1)

	for idx := range sp.Bins[:bins+1] {

		var vFreq = ((float64(idx) / float64(bins)) * cF) - cF
		vFreq = math.Pow(10.0, vFreq) * hi

		sp.Bins[idx].floorFFT = sp.freqToIdx(vFreq, math.Round)
		sp.Bins[idx].eqVal = math.Log2(float64(idx)+2) * cCoef

		if idx > 0 {
			if sp.Bins[idx-1].floorFFT >= sp.Bins[idx].floorFFT {
				sp.Bins[idx].floorFFT = sp.Bins[idx-1].floorFFT + 1
			}

			sp.Bins[idx-1].ceilFFT = sp.Bins[idx].floorFFT
		}
	}
}

func (sp *Spectrum) distributeEqual(bins int) {
	var loF = Frequencies[0]
	var hiF = math.Min(Frequencies[4], sp.SampleRate/2)
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

	for idx := range sp.Bins[:bins] {
		sp.Bins[idx].floorFFT = start
		start += spread

		sp.Bins[idx].ceilFFT = start
	}

	if last > 0 {
		sp.Bins[lBins].floorFFT = start
		sp.Bins[lBins].ceilFFT = start + last
	}
}

func (sp *Spectrum) idxToFreq(bin int) float64 {
	return float64(bin) * sp.SampleRate / float64(sp.SampleSize)
}

type mathFunc func(float64) float64

func (sp *Spectrum) freqToIdx(freq float64, round mathFunc) int {
	var b = int(round(freq / (sp.SampleRate / float64(sp.SampleSize))))

	if b < sp.fftSize {
		return b
	}

	return sp.fftSize - 1
}

// SetType will set the spectrum type
func (sp *Spectrum) SetType(st SpectrumType) {
	sp.sType = st
}

// SetWinVar sets the winVar used for distribution spread
func (sp *Spectrum) SetWinVar(g float64) {
	if g <= 0.0 {
		sp.winVar = 1.0
		return
	}

	sp.winVar = g
}

// SetSmoothing sets the smoothing parameters
func (sp *Spectrum) SetSmoothing(factor float64) {
	if factor <= 0.0 {
		factor = math.SmallestNonzeroFloat64
	}

	sp.smoothFactor = factor

	var sf = math.Pow(10.0, (1.0-factor)*(-25.0))

	sp.smoothPow = math.Pow(sf, float64(sp.SampleSize)/sp.SampleRate)
}
