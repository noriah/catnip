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
	numBars int

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

	winMax := int((AutoScalingSeconds*s.sampleRate)/float64(s.sampleSize)) * 2

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
	var bars int = s.numBars * s.frameSize
	return s.workBuffer[:bars:bars]
}

// Recalculate rebuilds our frequency bins with bars bin counts
func (s *Spectrum) Recalculate(bars int, lo, hi float64) int {
	if bars > s.maxBars {
		bars = s.maxBars
	}

	s.numBars = bars

	var (
		bax  int     // bar position index (left to right on screen)
		freq float64 // frequency for bar

		bins float64 // float of total number of bins we are using

		freqConst float64

		hRate float64
		qSize float64
	)

	bins = float64(s.numBars + 1.0)

	freqConst = math.Log10(lo/hi) / ((1.0 / bins) - 1.0)

	hRate = s.sampleRate / 2.0
	qSize = float64(s.sampleSize) / 4.0

	for bax = 0; bax <= s.numBars; bax++ {
		freq = (freqConst * -1) + ((float64(bax+1) / bins) * freqConst)
		freq = hi * math.Pow(10.0, freq)
		freq = freq / hRate
		freq = freq / qSize

		s.loCuts[bax] = binType(math.Floor(freq))

		if bax > 0 {
			if s.loCuts[bax] <= s.loCuts[bax-1] {
				s.loCuts[bax] = s.loCuts[bax-1] + 1
			}

			s.hiCuts[bax-1] = s.loCuts[bax-1]
		}
	}

	return s.numBars
}

// Generate makes numBars and dumps them in the buffer
func (s *Spectrum) Generate(buf fftw.CmplxBuffer) {

	var (

		// Indexes

		bax   int // Bar Index
		chx   int // Channel Index
		chidx int // Buffer Channel Index

		bl int // Number of samples in buf

		cut    binType // Frequency Cut
		barMag float64 // Frequency Magnitude
		boost  float64 // Boost Factor
	)

	bl = len(buf) / s.frameSize

	for chx = 0; chx < s.frameSize; chx++ {
		s.peakHeight[chx] = 0.125
	}

	for chx = 0; chx < s.frameSize; chx++ {

		for bax = 0; bax < s.numBars; bax++ {
			chidx = (bax * s.frameSize) + chx

			boost = math.Log2(float64(2+bax)) * (100.0 / float64(s.numBars))

			barMag = 0

			for cut = s.loCuts[bax]; cut <= s.hiCuts[bax] && cut < bl; cut++ {
				barMag += pyt(buf[(cut*s.frameSize)+chx])
			}

			barMag = barMag / float64(s.hiCuts[bax]-s.loCuts[bax]+1)
			barMag = barMag * boost

			s.workBuffer[chidx] = math.Pow(barMag, 0.5)

			if s.workBuffer[chidx] > s.peakHeight[chx] {
				s.peakHeight[chx] = s.workBuffer[chidx]
			}
		}
	}
}

func pyt(val fftw.CmplxType) float64 {
	return math.Sqrt((real(val) * real(val)) + (imag(val) * imag(val)))
}

// if bax > 0 {

// 	for pass = bax - 1; pass >= 0; pass-- {
// 		tmp = s.workBuffer[chidx] / math.Pow(factor, float64(bax-pass))
// 		if tmp > s.workBuffer[chidx] {
// 			s.workBuffer[chidx] = tmp
// 		}
// 	}

// 	for pass = bax + 1; pass < s.numBars; pass++ {
// 		tmp = s.workBuffer[chidx] / math.Pow(factor, float64(pass-bax))
// 		if tmp > s.workBuffer[chidx] {
// 			s.workBuffer[chidx] = tmp
// 		}
// 	}
// }

// for chx = 0; chx < s.frameSize; chx++ {

// 	barMag, stddev = s.heightWindow[chx].Update(s.peakHeight[chx])

// 	s.peakHeight[chx] = math.Max(barMag+(2*stddev), 1)

// 	for bax = 0; bax < s.numBars; bax++ {
// 		chidx = (bax * s.frameSize) + chx

// 		tmp = (s.workBuffer[chidx] / s.peakHeight[chx]) * boost

// 		s.barBuffer[chidx] = BarType(tmp)

// 	}
// }
