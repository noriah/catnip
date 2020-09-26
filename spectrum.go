package tavis

import (
	"fmt"
	"math"

	"github.com/noriah/tavis/fftw"
)

type BarType = int32

type BarBuffer []BarType

type FreqType = float64

type FreqBuffer []FreqType

type BinType = int

type BinBuffer []BinType

type Spectrum struct {
	max  int
	bars int

	SampleSize int
	SampleRate FreqType
	FrameSize  int

	BarBuffer BarBuffer

	loCutBins BinBuffer
	midBins   BinBuffer
	hiCutBins BinBuffer
	relBins   BinBuffer
	eqFreqs   FreqBuffer
}

// Init will set up our spectrum
func (s *Spectrum) Init() error {
	s.max = len(s.BarBuffer) / s.FrameSize
	s.loCutBins = make(BinBuffer, s.max+1)
	s.midBins = make(BinBuffer, s.max+1)
	s.hiCutBins = make(BinBuffer, s.max+1)
	s.relBins = make(BinBuffer, s.max+1)
	s.eqFreqs = make(FreqBuffer, s.max+1)

	s.Recalculate(s.max, 10, s.SampleRate)

	return nil
}

func (s *Spectrum) Print() {
	fmt.Println(s.max, s.loCutBins, s.hiCutBins)
}

// Recalculate rebuilds our frequency bins with num bin counts
func (s *Spectrum) Recalculate(num int, lo, hi FreqType) int {
	if num > s.max {
		num = s.max
	}

	s.bars = num

	num *= s.FrameSize

	var (
		bins FreqType
		// bin  FreqType

		freqConst FreqType
		freq      FreqType

		// hRate BinType
		// qSize FreqType

		idx int
	)

	bins = float64(s.bars + 1)
	freqConst = FreqType(math.Log10(lo/hi) / (1/bins - 1))

	// hRate = BinType(s.SampleRate / 2.0)
	// qSize = float64(s.SampleSize) / 4.0

	for idx = 0; idx <= s.bars; idx++ {
		freq = hi * math.Pow(10, (-1*freqConst)+(freqConst*(float64(idx+1)/bins)))

		s.loCutBins[idx] = BinType(freq)
		fmt.Println(freq)

		if idx > 0 {
			if s.loCutBins[idx] <= s.loCutBins[idx-1] {
				s.loCutBins[idx] = s.loCutBins[idx-1] + 1
			}

			s.hiCutBins[idx-1] = s.loCutBins[idx]
		}

		// s.relBins[idx] = s.loCutBins[idx] / hRate

		// s.eqFreqs[idx] = math.Pow(float64(s.loCutBins[idx]), 1)

		if idx > 0 {
		}
	}

	return s.bars
}

// Generate makes the bars in buffer
func (s *Spectrum) Generate(buf fftw.CmplxBuffer) {
	var (
		idx     int
		chx     int
		num     int
		fftwVar fftw.CmplxType
		freqMag float64
		cut     BinType
		boost   float64
	)

	num = len(buf)

	for idx = 0; idx < s.bars; idx++ {
		boost = math.Log2(float64(2+idx)) * (100 / float64(s.bars))

		for chx = idx * s.FrameSize; chx < ((idx + 1) * s.FrameSize); chx++ {

			freqMag = 0

			for cut = s.loCutBins[idx]; cut <= s.hiCutBins[idx] && cut < num; cut++ {

				fftwVar = buf[chx]

				freqMag += math.Sqrt((real(fftwVar) * real(fftwVar)) +
					(imag(fftwVar) * imag(fftwVar)))

			}

			freqMag /= FreqType(s.hiCutBins[idx] - s.loCutBins[idx] + 1)
			freqMag *= boost

			fmt.Println(chx)
			s.BarBuffer[chx] = BarType(math.Pow(freqMag, 0.5))

		}
	}
}

func (s *Spectrum) Waves(waves int) {
	if waves <= 0 {
		return
	}
}

func (s *Spectrum) Monstercat(factor float64) {
	if factor <= 1 {
		return
	}

	var (
		pass int
		bar  int
		barV BarType
	)

	for pass = 0; pass < s.bars; pass++ {
		for bar = pass - 1; bar >= 0; bar-- {
			barV = s.BarBuffer[pass] / pow(factor, pass-bar)
			if barV > s.BarBuffer[bar] {
				s.BarBuffer[bar] = barV
			}
		}

		for bar = pass + 1; bar < s.bars; bar++ {
			barV = s.BarBuffer[pass] / pow(factor, bar-pass)
			if barV > s.BarBuffer[bar] {
				s.BarBuffer[bar] = barV
			}
		}
	}
}

func max(bar, baz BarType) BarType {
	if bar < baz {
		return baz
	}
	return bar
}

func pow(factor float64, delta int) BarType {
	return BarType(math.Pow(factor, float64(delta)))
}
