package tavis

import (
	"math"

	"github.com/noriah/tavis/fftw"
)

// Spectrum Constants
const (
	AutoScalingSeconds = 10

	AutoScalingDumpPercent = 0.75

	MaxBars = 1024
)

// DataSet represents a channel or sample index in a series frame
type DataSet struct {
	id int

	Data    []float64
	falloff []float64

	peakHeight float64
	window     *MovingWindow
}

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBins int
	numBins int

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	sampleDataSize int

	// frameSize is the number of channels we expect per frame
	frameSize int

	// winMax is the maximum number of values in our sliding window
	winMax int

	// DataBuf is a slice of fftw.ComplexType values
	DataBuf []fftw.ComplexType

	// workSets is a slice of float64 values
	workSets []*DataSet

	loCuts []int
	hiCuts []int
}

// Init will set up our spectrum
func (s *Spectrum) Init() error {

	s.maxBins = MaxBars

	s.workSets = make([]*DataSet, s.frameSize)

	s.winMax = int((AutoScalingSeconds*s.sampleRate)/float64(s.sampleSize)) * 2

	for idx := 0; idx < s.frameSize; idx++ {
		s.workSets[idx] = &DataSet{
			id:      idx,
			Data:    make([]float64, s.maxBins),
			falloff: make([]float64, s.maxBins),
			window:  NewMovingWindow(s.winMax),
		}
	}

	s.loCuts = make([]int, s.maxBins+1)
	s.hiCuts = make([]int, s.maxBins+1)

	s.Recalculate(20, 20, s.sampleRate/2)

	return nil
}

// DataSets returns our sets of data
func (s *Spectrum) DataSets() []*DataSet {
	return s.workSets
}

// Recalculate rebuilds our frequency bins with bins bin counts
func (s *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > s.maxBins {
		bins = s.maxBins
	}

	s.numBins = bins

	var (
		cBins float64 // bin count constant
		cFreq float64 // frequency step constant

		xBin int // bin index

		vFreq float64 // frequency variable
	)

	cBins = float64(s.numBins + 1)

	cFreq = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin = 0; xBin <= s.numBins; xBin++ {
		vFreq = (cFreq * -1) + ((float64(xBin+1) / cBins) * cFreq)
		vFreq = hi * math.Pow(10.0, vFreq)
		vFreq = (vFreq / (s.sampleRate / 2)) * (float64(s.sampleSize) / 4)

		s.loCuts[xBin] = int(math.Floor(vFreq))

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
func (s *Spectrum) Generate() {

	var (
		xBin  int // bin index
		xSet  int // set index
		xFreq int // frequency index

		vMag   float64 // Frequency Magnitude variable
		vBoost float64 // Boost Factor

	)

	for xSet = range s.workSets {

		for xBin = 0; xBin <= s.numBins+1; xBin++ {

			vBoost = math.Log2(float64(2+xBin)) * (100.0 / float64(s.numBins))

			vMag = 0

			for xFreq = s.loCuts[xBin]; xFreq <= s.hiCuts[xBin] &&
				xFreq < s.sampleDataSize; xFreq++ {
				vMag = vMag + pyt(s.DataBuf[(xFreq*s.frameSize)+xSet])
				// vMag = vMag + pyt(s.DataBuf[xFreq+(s.sampleDataSize*xSet)])
			}

			vMag = vMag / float64(s.hiCuts[xBin]-s.loCuts[xBin]+1)
			vMag = vMag * vBoost

			s.workSets[xSet].Data[xBin] = math.Pow(vMag, 0.5)
		}
	}
}

func pyt(value fftw.ComplexType) float64 {
	return math.Sqrt(float64((real(value) * real(value)) + (imag(value) * imag(value))))
}

// Scale scales the data
func (s *Spectrum) Scale(height int) {
	var (
		xBin int // bin index

		cHeight float64 // height constant

		vSet  *DataSet
		vMag  float64 // magnitude variable
		vMean float64 // average variable
		vSD   float64 // standard deviation variable
	)

	cHeight = float64(height)

	for _, vSet = range s.workSets {

		if vSet.window.Points() >= s.winMax {
			vSet.window.Drop(int(float64(s.winMax) * AutoScalingDumpPercent))
		}

		vSet.peakHeight = 0.125

		for xBin = 0; xBin <= s.numBins; xBin++ {
			if vSet.peakHeight < vSet.Data[xBin] {
				vSet.peakHeight = vSet.Data[xBin]
			}
		}

		vMean, vSD = vSet.window.Update(vSet.peakHeight)

		vMag = math.Max(vMean+(2*vSD), 1.0)

		for xBin = 0; xBin <= s.numBins; xBin++ {
			vSet.Data[xBin] = ((vSet.Data[xBin] / vMag) * cHeight) - 1

			vSet.Data[xBin] = math.Min(cHeight-1, vSet.Data[xBin])
		}
	}
}

// Monstercat is not entirely understood yet.
func (s *Spectrum) Monstercat(factor float64) {

	var (
		xBin int
		pass int
		vSet *DataSet
		tmp  float64
	)

	for _, vSet = range s.workSets {
		for xBin = 0; xBin <= s.numBins; xBin++ {
			if xBin > 0 {
				for pass = xBin - 1; pass >= 0; pass-- {
					tmp = vSet.Data[xBin] / math.Pow(factor, float64(xBin-pass))
					if tmp > vSet.Data[xBin] {
						vSet.Data[xBin] = tmp
					}
				}

				for pass = xBin + 1; pass <= s.numBins; pass++ {
					tmp = vSet.Data[xBin] / math.Pow(factor, float64(pass-xBin))
					if tmp > vSet.Data[xBin] {
						vSet.Data[xBin] = tmp
					}
				}
			}
		}
	}
}

// Falloff is a simple falloff function
func (s *Spectrum) Falloff(weight float64) {
	var (
		xBin int
		vMag float64
		vSet *DataSet
	)

	for _, vSet = range s.workSets {
		for xBin = 0; xBin <= s.numBins; xBin++ {
			vMag = vSet.falloff[xBin]
			vMag = math.Min(vMag*weight, vMag-1)
			vMag = math.Max(vMag, vSet.Data[xBin])
			vSet.falloff[xBin] = vMag
			vSet.Data[xBin] = vMag
		}
	}
}
