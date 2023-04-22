// Package dsp provides audio analysis
//
// Some notes:
//
// https://dlbeer.co.nz/articles/fftvis.html
// https://www.cg.tuwien.ac.at/courses/WissArbeiten/WS2010/processing.pdf
// https://github.com/hvianna/audioMotion-analyzer/blob/master/src/audioMotion-analyzer.js#L1053
// https://dsp.stackexchange.com/questions/6499/help-calculating-understanding-the-mfccs-mel-frequency-cepstrum-coefficients
// https://stackoverflow.com/questions/3694918/how-to-extract-frequency-associated-with-fft-values-in-python
//   - https://stackoverflow.com/a/27191172
package dsp

import "math"

type BinMethod func(int, float64, float64) float64

type AnalyzerConfig struct {
	SampleRate    float64   // audio sample rate
	SampleSize    int       // number of samples per slice
	SquashLow     bool      // squash the low end the spectrum
	SquashLowOld  bool      // squash the low end using the old method
	DontNormalize bool      // dont run math.Log on output
	BinMethod     BinMethod // method used for calculating bin value
}

type Analyzer interface {
	BinCount() int
	ProcessBin(int, []complex128) float64
	Recalculate(int) int
}

// analyzer is an audio spectrum in a buffer
type analyzer struct {
	cfg      AnalyzerConfig // the analyzer config
	bins     []bin          // bins for processing
	binCount int            // number of bins we look at
	fftSize  int            // number of fft bins
}

// Bin is a helper struct for spectrum
type bin struct {
	powVal   float64 // powpow
	eqVal    float64 // equalizer value
	floorFFT int     // floor fft index
	ceilFFT  int     // ceiling fft index
	// widthFFT int     // fft floor-ceiling index delta
}

// frequencies are the dividing frequencies
var frequencies = []float64{
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

// Average all the samples together.
func AverageSamples() BinMethod {
	return func(count int, current, new float64) float64 {
		return current + (new / float64(count))
	}
}

// Sum all the samples together.
func SumSamples() BinMethod {
	return func(_ int, current, new float64) float64 {
		return current + new
	}
}

// Return the maximum value of all the samples.
func MaxSampleValue() BinMethod {
	return func(_ int, current, new float64) float64 {
		if current < new {
			return new
		}
		return current
	}
}

// Return the minimum value of all the samples that is not zero.
func MinNonZeroSampleValue() BinMethod {
	return func(_ int, current, new float64) float64 {
		if current == 0.0 {
			return new
		}

		if new == 0.0 {
			return current
		}

		if current > new {
			return new
		}

		return current
	}
}

func NewAnalyzer(cfg AnalyzerConfig) Analyzer {
	return &analyzer{
		cfg:     cfg,
		bins:    make([]bin, cfg.SampleSize),
		fftSize: cfg.SampleSize/2 + 1,
	}
}

// BinCount returns the number of bins each stream has
func (az *analyzer) BinCount() int {
	return az.binCount
}

func (az *analyzer) ProcessBin(idx int, src []complex128) float64 {
	bin := az.bins[idx]

	fftFloor, fftCeil := bin.floorFFT, bin.ceilFFT
	if fftCeil > az.fftSize {
		fftCeil = az.fftSize
	}

	src = src[fftFloor:fftCeil]
	mag := 0.0
	count := len(src)
	for _, cmplx := range src {
		power := math.Hypot(real(cmplx), imag(cmplx))
		mag = az.cfg.BinMethod(count, mag, power)
	}

	if az.cfg.SquashLow {
		// squash the low low end a bit.
		if az.cfg.SquashLowOld {
			if f := az.freqToIdx(400.0, math.Floor); fftFloor < f {
				mag *= 0.55 * (float64(fftFloor+1) / float64(f))
			}
		} else {
			if f := az.freqToIdx(600.0, math.Floor); fftFloor < f {
				mag *= 0.55 * (math.Min(1.0, (float64(fftFloor+1) / float64(f))))
			}
		}
	}

	if mag < 0.0 {
		return 0.0
	}

	if !az.cfg.DontNormalize {
		mag = math.Log(mag)
	}

	if mag < 0.0 {
		return 0.0
	}

	return mag
}

// Recalculate rebuilds our frequency bins
func (az *analyzer) Recalculate(binCount int) int {
	if az.fftSize == 0 {
		az.fftSize = az.cfg.SampleSize/2 + 1
	}

	switch {
	case binCount >= az.fftSize:
		binCount = az.fftSize - 1
	case binCount == az.binCount:
		return binCount
	}

	az.binCount = binCount

	// clean the binCount
	for idx := range az.bins[:binCount] {
		az.bins[idx].powVal = 0.65
		az.bins[idx].eqVal = 1.0
	}

	az.distribute(binCount)

	bassCut := az.freqToIdx(frequencies[2], math.Floor)
	fBassCut := float64(bassCut)

	// set widths
	for idx, b := range az.bins[:binCount] {
		if b.ceilFFT >= az.fftSize {
			az.bins[idx].ceilFFT = az.fftSize - 1
		}

		// az.bins[idx].widthFFT = b.ceilFFT - b.floorFFT

		if b.ceilFFT <= bassCut {
			az.bins[idx].powVal *= math.Max(0.5, float64(b.ceilFFT)/fBassCut)
		}

	}

	return binCount
}

// This is some hot garbage.
// It essentially is a lot of work to just increment from 0 for each next bin.
// Working on replacing this with a real distribution.
func (az *analyzer) distribute(bins int) {
	lo := frequencies[1]
	hi := math.Min(az.cfg.SampleRate/2, frequencies[4])

	loLog := math.Log10(lo)
	hiLog := math.Log10(hi)

	cF := (hiLog - loLog) / float64(bins)

	cCoef := 100.0 / float64(bins+1)

	for idx := range az.bins[:bins+1] {

		frequency := ((float64(idx) * cF) + loLog)
		frequency = math.Pow(10.0, frequency)
		fftIdx := az.freqToIdx(frequency, math.Floor)
		az.bins[idx].floorFFT = fftIdx
		az.bins[idx].eqVal = math.Log2(float64(fftIdx)+14) * cCoef
		// az.bins[idx].eqVal = 1.0

		if idx > 0 {
			if az.bins[idx-1].floorFFT >= az.bins[idx].floorFFT {
				az.bins[idx].floorFFT = az.bins[idx-1].floorFFT + 1
			}

			az.bins[idx-1].ceilFFT = az.bins[idx].floorFFT
		}
	}
}

type mathFunc func(float64) float64

func (az *analyzer) freqToIdx(freq float64, round mathFunc) int {
	b := int(round(freq / (az.cfg.SampleRate / float64(az.cfg.SampleSize))))

	if b < az.fftSize {
		return b
	}

	return az.fftSize - 1
}
