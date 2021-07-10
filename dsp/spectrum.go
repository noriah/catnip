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

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	Bins         []Bin       // bins for processing
	SampleSize   int         // number of samples per slice
	binCount     int         // number of bins we look at
	fftSize      int         // number of fft bins
	OldValues    [][]float64 // old values used for smoothing
	SampleRate   float64     // audio sample rate
	winVar       float64     // window variable
	smoothFactor float64     // smothing factor
	smoothScale  float64     // smoothing pow
}

// Bin is a helper struct for spectrum
type Bin struct {
	powVal   float64 // powpow
	eqVal    float64 // equalizer value
	floorFFT int     // floor fft index
	ceilFFT  int     // ceiling fft index
	// widthFFT int     // fft floor-ceiling index delta
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
	8000.0, // 4
	// brilliance
	22050.0, // 5
	// everything else
}

// BinCount returns the number of bins each stream has
func (sp *Spectrum) BinCount() int {
	return sp.binCount
}

func (sp *Spectrum) ProcessBin(ch, idx int, src []complex128) float64 {
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

	// time smoothing
	mag = math.Pow(mag, bin.powVal) * (1.0 - sp.smoothScale)

	value := (sp.OldValues[ch][idx] * sp.smoothScale) + mag
	sp.OldValues[ch][idx] = value

	return value * bin.eqVal
}

// Recalculate rebuilds our frequency bins
func (sp *Spectrum) Recalculate(binCount int) int {
	if sp.fftSize == 0 {
		sp.fftSize = sp.SampleSize/2 + 1
	}

	switch {
	case binCount >= sp.fftSize:
		binCount = sp.fftSize - 1
	case binCount == sp.binCount:
		return binCount
	}

	sp.binCount = binCount

	// clean the binCount
	for idx := range sp.Bins[:binCount] {
		sp.Bins[idx] = Bin{
			powVal: 0.65,
			eqVal:  1.0,
		}
	}

	sp.distribute(binCount)

	var bassCut = sp.freqToIdx(Frequencies[2], math.Floor)
	var fBassCut = float64(bassCut)

	// set widths
	for idx, b := range sp.Bins[:binCount] {
		if b.ceilFFT >= sp.fftSize {
			sp.Bins[idx].ceilFFT = sp.fftSize - 1
		}

		// sp.Bins[idx].widthFFT = b.ceilFFT - b.floorFFT

		if b.ceilFFT <= bassCut {
			sp.Bins[idx].powVal *= math.Max(0.5, float64(b.ceilFFT)/fBassCut)
		}

	}

	return binCount
}

func (sp *Spectrum) distribute(bins int) {
	var lo = Frequencies[1]
	var hi = math.Min(sp.SampleRate/2, Frequencies[4])

	var loLog = math.Log10(lo)
	var hiLog = math.Log10(hi)

	var cF = (hiLog - loLog) / float64(bins)

	var cCoef = 100.0 / float64(bins+1)

	for idx := range sp.Bins[:bins+1] {

		frequency := ((float64(idx) * cF) + loLog)
		frequency = math.Pow(10.0, frequency)
		fftIdx := sp.freqToIdx(frequency, math.Floor)
		sp.Bins[idx].floorFFT = fftIdx
		sp.Bins[idx].eqVal = math.Log2(float64(fftIdx)+14) * cCoef
		// sp.Bins[idx].eqVal = 1.0

		if idx > 0 {
			if sp.Bins[idx-1].floorFFT >= sp.Bins[idx].floorFFT {
				sp.Bins[idx].floorFFT = sp.Bins[idx-1].floorFFT + 1
			}

			sp.Bins[idx-1].ceilFFT = sp.Bins[idx].floorFFT
		}
	}
}

type mathFunc func(float64) float64

func (sp *Spectrum) freqToIdx(freq float64, round mathFunc) int {
	var b = int(round(freq / (sp.SampleRate / float64(sp.SampleSize))))

	if b < sp.fftSize {
		return b
	}

	return sp.fftSize - 1
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

	sp.smoothScale = math.Pow(sf, float64(sp.SampleSize)/sp.SampleRate)
}
