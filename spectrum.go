package tavis

import (
	"math"
)

// Spectrum Constants
const (
	// ScalingFastWindow in seconds
	ScalingSlowWindow = 10

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.1

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	ScalingResetDeviation = 1.0

	MaxBars = 1024
)

// DataSet represents a channel or sample index in a series frame
type DataSet struct {
	id int

	dataSize int
	numBins  int

	DataBuf []complex128
	binBuf  []float64
	prevBuf []float64

	peakHeight float64

	slowWindow *MovingWindow
	fastWindow *MovingWindow
}

// Bins returns the bins that we have as a silce
func (ds *DataSet) Bins() []float64 {
	return ds.binBuf[:ds.numBins]
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

	// dataSets is a slice of float64 values
	dataSets []*DataSet

	loCuts []int
	hiCuts []int
}

// NewSpectrum will set up our spectrum
func NewSpectrum(rate float64, sets, size int) *Spectrum {

	var s = &Spectrum{
		frameSize:  sets,
		sampleSize: size,
		sampleRate: rate,
	}

	s.maxBins = MaxBars

	slowMax := int((ScalingSlowWindow*s.sampleRate)/float64(s.sampleSize)) * 2
	fastMax := int((ScalingFastWindow*s.sampleRate)/float64(s.sampleSize)) * 2

	s.dataSets = make([]*DataSet, s.frameSize)

	dataSize := (size / 2) + 1

	for idx := 0; idx < s.frameSize; idx++ {
		s.dataSets[idx] = &DataSet{
			id:       idx,
			dataSize: dataSize,
			// DataBuf:    make([]complex128, dataSize),
			binBuf:     make([]float64, s.maxBins),
			prevBuf:    make([]float64, s.maxBins),
			slowWindow: NewMovingWindow(slowMax),
			fastWindow: NewMovingWindow(fastMax),
		}
	}

	s.loCuts = make([]int, s.maxBins+1)
	s.hiCuts = make([]int, s.maxBins+1)

	s.Recalculate(s.maxBins, 20, s.sampleRate/2)

	return s
}

// DataSets returns our sets of data
func (s *Spectrum) DataSets() []*DataSet {
	return s.dataSets
}

// Recalculate rebuilds our frequency bins with bins bin counts
//
// reference: https://github.com/karlstav/cava/blob/master/cava.c#L654
// reference: https://github.com/noriah/cli-visualizer/blob/master/src/Transformer/SpectrumTransformer.cpp#L598
func (s *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > s.maxBins {
		bins = s.maxBins
	}

	for _, vSet := range s.dataSets {
		vSet.numBins = bins
	}

	s.numBins = bins

	var cBins = float64(bins + 1)

	var cFreq = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin := 0; xBin <= bins; xBin++ {
		vFreq := (((float64(xBin+1) / cBins) - 1) * cFreq)
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

	for _, vS := range s.dataSets {

		for xBin := 0; xBin <= vS.numBins; xBin++ {

			var vM = 0.0

			for xF, vC := s.loCuts[xBin], complex128(0); xF <= s.hiCuts[xBin] &&
				xF < vS.dataSize; xF++ {

				vC = vS.DataBuf[xF]

				vM += math.Sqrt(
					(real(vC) * real(vC)) +
						(imag(vC) * imag(vC)))
			}

			vM /= float64(s.hiCuts[xBin] - s.loCuts[xBin] + 1)

			vM *= (math.Log2(float64(2+xBin)) *
				(100.0 / float64(vS.numBins)))

			vS.binBuf[xBin] = math.Pow(vM, 0.5)
		}
	}
}

// Scale scales the data
func (s *Spectrum) Scale(height int) {
	var (
		xBin int // bin index

		vMag float64 // magnitude variable
	)

	var cHeight = float64(height)

	for _, vSet := range s.dataSets {

		vSet.peakHeight = 0.125
		var vSilent = true

		for xBin = 0; xBin <= s.numBins; xBin++ {
			if vSet.binBuf[xBin] > 0 {
				vSilent = false
				if vSet.peakHeight < vSet.binBuf[xBin] {
					vSet.peakHeight = vSet.binBuf[xBin]
				}
			}
		}

		if vSilent {
			return
		}

		vSet.fastWindow.Update(vSet.peakHeight)
		var vMean, vSD = vSet.slowWindow.Update(vSet.peakHeight)

		if xBin = vSet.slowWindow.Len(); xBin > vSet.fastWindow.Cap() {
			vMag = math.Abs(vSet.fastWindow.Mean() - vMean)
			if vMag > (ScalingResetDeviation * vSD) {
				vSet.slowWindow.Drop(int(float64(xBin) * ScalingDumpPercent))
				vMean, vSD = vSet.slowWindow.Stats()
			}
		}

		vMag = math.Max(vMean+(2*vSD), 1.0)

		for xBin = 0; xBin <= s.numBins; xBin++ {
			vSet.binBuf[xBin] = ((vSet.binBuf[xBin] / vMag) * cHeight) - 1

			vSet.binBuf[xBin] = math.Min(cHeight-1, vSet.binBuf[xBin])
		}
	}
}

// Monstercat is not entirely understood yet.
// We need to work on it
func (s *Spectrum) Monstercat(factor float64) {

	for _, vSet := range s.dataSets {

		for xBin := 1; xBin <= vSet.numBins; xBin++ {

			for xPass := 0; xPass <= vSet.numBins; xPass++ {

				tmp := vSet.binBuf[xBin] / math.Pow(factor, absInt(xBin-xPass))

				if tmp > vSet.binBuf[xBin] {
					vSet.binBuf[xBin] = tmp
				}
			}
		}
	}
}

func absInt(value int) float64 {
	if value < 0 {
		return float64(-value)
	}
	return float64(value)
}

// Falloff does falling off things
func (s *Spectrum) Falloff(weight float64) {

	for _, vSet := range s.dataSets {
		for xBin := 0; xBin <= s.numBins; xBin++ {
			vMag := vSet.prevBuf[xBin]
			vMag = math.Min(vMag*weight, vMag-1)

			// we want the higher value here because we just calculated the
			// lower value without checking if we need it
			vMag = math.Max(vMag, vSet.binBuf[xBin])
			vSet.prevBuf[xBin] = vMag
			vSet.binBuf[xBin] = vMag
		}
	}
}
