package tavis

import (
	"fmt"
	"math"

	"github.com/noriah/tavis/fftw"
	"github.com/noriah/tavis/util"
)

// Spectrum Constants
const (
	AutoScalingSeconds = 10
)

// BarType is the type of each bar value
type BarType = float64

// BarBuffer is a slice of bar types
type BarBuffer []BarType

// ScratchType is a type of each frequency value
type ScratchType = float64

// ScratchBuffer is a slice of ScratchType
type ScratchBuffer []ScratchType

// BinType is a type of each bin cutoff value
type BinType = int

// BinBuffer is aslice of BinType
type BinBuffer []BinType

// Spectrum creates an audio spectrum in a buffer
type Spectrum struct {
	max  int
	bars int

	SampleSize int
	SampleRate ScratchType
	FrameSize  int

	Data fftw.CmplxBuffer

	BarBuffer BarBuffer

	workBuffer ScratchBuffer

	heightWindow []*util.MovingWindow
	maxHeights   ScratchBuffer

	loCutBins BinBuffer
	hiCutBins BinBuffer
}

// Init will set up our spectrum
func (s *Spectrum) Init() error {
	s.max = len(s.BarBuffer) / s.FrameSize

	s.workBuffer = make(ScratchBuffer, s.max+1)

	winMax := int((AutoScalingSeconds*s.SampleRate)/float64(s.SampleSize)) * 2

	s.heightWindow = make([]*util.MovingWindow, s.FrameSize)
	s.maxHeights = make(ScratchBuffer, s.FrameSize)

	for idx := 0; idx < s.FrameSize; idx++ {
		s.heightWindow[idx] = util.NewMovingWindow(winMax)
	}

	s.loCutBins = make(BinBuffer, s.max+1)
	s.hiCutBins = make(BinBuffer, s.max+1)

	s.Recalculate(s.max, 20, s.SampleRate/2)

	return nil
}

// Print is a debug function to print internal structure.
// Will be removed later.
func (s *Spectrum) Print() {
	fmt.Println(s.max, s.loCutBins, s.hiCutBins)
}

// Recalculate rebuilds our frequency bins with num bin counts
func (s *Spectrum) Recalculate(num int, lo, hi ScratchType) int {
	if num > s.max {
		num = s.max
	}

	s.bars = num

	var (
		bins ScratchType

		freqConst ScratchType
		freq      ScratchType

		hRate ScratchType
		qSize ScratchType

		idx int
	)

	hRate = s.SampleRate / 2
	qSize = float64(s.SampleSize) / 2

	bins = float64(s.bars + 1)
	freqConst = ScratchType(math.Log10(lo/hi) / ((1 / bins) - 1))

	for idx = 0; idx <= s.bars; idx++ {
		freq = hi * math.Pow(10, (freqConst*-1)+((float64(idx+1)/bins)*freqConst))

		s.loCutBins[idx] = BinType((freq / hRate) * qSize)

		if idx > 0 {
			if s.loCutBins[idx] <= s.loCutBins[idx-1] {
				s.loCutBins[idx] = s.loCutBins[idx-1] + 1
			}

			s.hiCutBins[idx-1] = s.loCutBins[idx] - 1
		}
	}

	return s.bars
}

// Generate makes bars and dumps them in the BarBuffer
// Look at all the loops.
// Someone stop me!
func (s *Spectrum) Generate(height int, factor float64) {

	var (

		// Indexes

		bufLen int // Number of samples in s.Data
		bax    int // Bar Index
		chx    int // Channel Index
		chidx  int // Buffer Channel Index

		// Monstercat

		pass int         // Pass number for monstercat
		tmp  ScratchType // tmp value for monstercat

		fftwVar fftw.CmplxType // FFTW Value
		freqMag float64        // Frequency Magnitude

		cut    BinType // Frequency Cut
		boost  float64 // Boost Factor
		stddev float64 // Standard Deviation for scaling
	)

	bufLen = len(s.Data)

	for chx = 0; chx < s.FrameSize; chx++ {
		s.maxHeights[chx] = 0.125
	}

	for bax = 0; bax < s.bars; bax++ {
		boost = math.Log2(float64(2+bax)) * (100 / float64(s.bars))

		for chx = 0; chx < s.FrameSize; chx++ {
			chidx = (bax * s.FrameSize) + chx

			freqMag = 0

			for cut = s.loCutBins[bax]; cut <= s.hiCutBins[bax] && cut < bufLen; cut++ {

				fftwVar = s.Data[chidx]

				freqMag += math.Sqrt((real(fftwVar) * real(fftwVar)) +
					(imag(fftwVar) * imag(fftwVar)))

			}

			freqMag /= ScratchType(s.hiCutBins[bax] - s.loCutBins[bax] + 1)
			freqMag *= boost

			s.workBuffer[chidx] = math.Pow(freqMag, 0.5)

			if bax > 0 {

				for pass = bax - 1; pass >= 0; pass-- {
					tmp = s.workBuffer[chidx] / math.Pow(factor, float64(bax-pass))
					if tmp > s.workBuffer[chidx] {
						s.workBuffer[chidx] = tmp
					}
				}

				for pass = bax + 1; pass < s.bars; pass++ {
					tmp = s.workBuffer[chidx] / math.Pow(factor, float64(pass-bax))
					if tmp > s.workBuffer[chidx] {
						s.workBuffer[chidx] = tmp
					}
				}
			}

			if s.workBuffer[chidx] > s.maxHeights[chx] {
				s.maxHeights[chx] = s.workBuffer[chidx]
			}
		}
	}

	boost = float64(height)

	for chx = 0; chx < s.FrameSize; chx++ {

		freqMag, stddev = s.heightWindow[chx].Update(s.maxHeights[chx])

		s.maxHeights[chx] = math.Max(freqMag+(2*stddev), 1)

		for bax = 0; bax < s.bars; bax++ {
			chidx = (bax * s.FrameSize) + chx

			s.BarBuffer[chidx] = BarType((s.workBuffer[chidx] / s.maxHeights[chx]) * boost)

		}
	}
}
