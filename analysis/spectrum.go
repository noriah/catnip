package analysis

import (
	"fmt"
	"math"

	"github.com/noriah/tavis/analysis/fftw"
	"github.com/noriah/tavis/util"
)

// Spectrum Constants
const (
	AutoScalingSeconds = 10

	MaxBars = 1024
)

// binType is a type of each bin cutoff value
type binType = int

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBars int
	numBins int

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	// frameSize is the number of channels we expect per frame
	frameSize int

	workBuffer []float64

	heightWindow []*util.MovingWindow
	peakHeight   []float64

	loCuts []binType
	hiCuts []binType
}

// NewSpectrum returns a new Spectrum
func NewSpectrum(rate float64, samples, frame int) *Spectrum {
	var newSpectrum *Spectrum = &Spectrum{
		sampleRate: rate,
		sampleSize: samples,
		frameSize:  frame,
	}

	if err := newSpectrum.init(); err != nil {
		panic(err)
	}

	return newSpectrum
}

// Init will set up our spectrum
func (s *Spectrum) init() error {

	s.maxBars = MaxBars

	s.workBuffer = make([]float64, s.maxBars*s.frameSize)

	winMax := int((AutoScalingSeconds * s.sampleRate) / float64(s.sampleSize))

	s.heightWindow = make([]*util.MovingWindow, s.frameSize)
	s.peakHeight = make([]float64, s.frameSize)

	for idx := 0; idx < s.frameSize; idx++ {
		s.heightWindow[idx] = util.NewMovingWindow(winMax)
	}

	s.loCuts = make([]binType, s.maxBars+1)
	s.hiCuts = make([]binType, s.maxBars+1)

	s.Recalculate(s.maxBars, 20, s.sampleRate/2)

	return nil
}

// Print is a debug function to print internal structure.
// Will be removed later.
func (s *Spectrum) Print() {
	fmt.Println(s.maxBars, s.loCuts, s.hiCuts)
}

// Bins returns a slice of only the bins we are expecting to process
func (s *Spectrum) Bins() []float64 {
	var bars int = s.numBins * s.frameSize
	return s.workBuffer[:bars:bars]
}

// Recalculate rebuilds our frequency bins with bars bin counts
func (s *Spectrum) Recalculate(bars int, lo, hi float64) int {
	if bars > s.maxBars {
		bars = s.maxBars
	}

	s.numBins = bars

	var (
		cBins float64 // bin count constant
		cFreq float64 // frequency step constant

		xBin int // bin index

		vFreq float64 // frequency variable
	)

	cBins = float64(s.numBins + 1.0)

	cFreq = math.Log10(lo/hi) / ((1.0 / cBins) - 1.0)

	for xBin = 0; xBin <= s.numBins; xBin++ {
		vFreq = (cFreq) + ((float64(xBin+1) / cBins) * cFreq)
		vFreq = hi * math.Pow(10.0, vFreq)
		vFreq = (vFreq / (s.sampleRate / 2.0)) / (float64(s.sampleSize) / 4.0)

		s.loCuts[xBin] = binType(math.Floor(vFreq))

		if xBin > 0 {
			if s.loCuts[xBin] <= s.loCuts[xBin-1] {
				s.loCuts[xBin] = s.loCuts[xBin-1] + 1
			}

			s.hiCuts[xBin-1] = s.loCuts[xBin-1]
		}
	}

	return s.numBins
}

// Generate makes numBins and dumps them in the buffer
func (s *Spectrum) Generate(buf fftw.CmplxBuffer) {

	var (

		// Indexes

		xBin int // bin index
		xChn int // channel Index
		xBuf int // buffer Index

		xFreq binType // frequency index

		bl int // Number of samples in buf

		barMag float64 // Frequency Magnitude
		boost  float64 // Boost Factor
	)

	bl = len(buf) / s.frameSize

	for xChn = 0; xChn < s.frameSize; xChn++ {
		s.peakHeight[xChn] = 1
	}

	for xChn = 0; xChn < s.frameSize; xChn++ {

		for xBin = 0; xBin < s.numBins; xBin++ {
			xBuf = (xBin * s.frameSize) + xChn

			boost = math.Log2(float64(2+xBin)) * (100.0 / float64(s.numBins))

			barMag = 0

			for xFreq = s.loCuts[xBin]; xFreq <= s.hiCuts[xBin] && xFreq < binType(bl); xFreq++ {
				barMag = barMag + pyt(buf[(int(xFreq)*s.frameSize)+xChn])
			}

			barMag = barMag / float64(s.hiCuts[xBin]-s.loCuts[xBin]+1)
			barMag = barMag * boost

			s.workBuffer[xBuf] = math.Pow(barMag, 0.5)

			if s.workBuffer[xBuf] > s.peakHeight[xChn] {
				s.peakHeight[xChn] = s.workBuffer[xBuf]
			}
		}
	}
}

func pyt(val fftw.CmplxType) float64 {
	return math.Sqrt((real(val) * real(val)) + (imag(val) * imag(val)))
}

// Scale scales the data
func (s *Spectrum) Scale(height int) {
	var (
		xBin  int // bin index
		xChan int // channel index

		cHeight float64 // window half height constant

		vMag  float64 // magnitude variable
		vMean float64 // average variable
		vSD   float64 // standard deviation variable
	)

	cHeight = float64(height)

	for xChan = 0; xChan < s.frameSize; xChan++ {
		vMean, vSD = s.heightWindow[xChan].Update(s.peakHeight[xChan])

		vMag = math.Max(vMean+(2*vSD), 1.0)

		for xBin = xChan; xBin < s.numBins*s.frameSize+xChan; xBin += s.frameSize {
			s.workBuffer[xBin] = math.Min(cHeight-1, ((s.workBuffer[xBin]/vMag)*cHeight)-1)
		}
	}
}

// func (s *Spectrum)

// if xBin > 0 {

// 	for pass = xBin - 1; pass >= 0; pass-- {
// 		tmp = s.workBuffer[xBuf] / math.Pow(factor, float64(xBin-pass))
// 		if tmp > s.workBuffer[xBuf] {
// 			s.workBuffer[xBuf] = tmp
// 		}
// 	}

// 	for pass = xBin + 1; pass < s.numBins; pass++ {
// 		tmp = s.workBuffer[xBuf] / math.Pow(factor, float64(pass-xBin))
// 		if tmp > s.workBuffer[xBuf] {
// 			s.workBuffer[xBuf] = tmp
// 		}
// 	}
// }

// for xChn = 0; xChn < s.frameSize; xChn++ {

// 	barMag, stddev = s.heightWindow[xChn].Update(s.peakHeight[xChn])

// 	s.peakHeight[xChn] = math.Max(barMag+(2*stddev), 1)

// 	for xBin = 0; xBin < s.numBins; xBin++ {
// 		xBuf = (xBin * s.frameSize) + xChn

// 		tmp = (s.workBuffer[xBuf] / s.peakHeight[xChn]) * boost

// 		s.barBuffer[xBuf] = BarType(tmp)

// 	}
// }
