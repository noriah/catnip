package tavis

import (
	"math"

	"github.com/noriah/tavis/fftw"
)

// Spectrum Constants
const (
	// ScalingFastWindow in seconds
	ScalingSlowWindow = 10

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	ScalingResetDeviation = 1

	MaxBins = 1024
)

// DataSet represents a channel or sample index in a series frame
type DataSet struct {
	id int

	spectrum *Spectrum

	fftwPlan *fftw.Plan

	dataBuf  []complex128
	dataSize int

	binBuf  []float64
	prevBuf []float64
	numBins int

	peakHeight float64

	slowWindow *MovingWindow
	fastWindow *MovingWindow
}

// ID returns the set id
func (ds *DataSet) ID() int {
	return ds.id
}

// Bins returns the bins that we have as a silce
func (ds *DataSet) Bins() []float64 {
	return ds.binBuf[:ds.numBins]
}

// Size returns the number of bins we have processed
func (ds *DataSet) Size() int {
	return ds.numBins
}

// ExecuteFFTW executes fftw math on the source buffer
func (ds *DataSet) ExecuteFFTW() {
	ds.fftwPlan.Execute()
}

// Spectrum is an audio spectrum in a buffer
type Spectrum struct {
	maxBins int
	numBins int

	setCount int

	dataSize int

	// sampleSize is the number of frames per sample
	sampleSize int

	// sampleRate is the frequency that samples are collected
	sampleRate float64

	loCuts []int
	hiCuts []int
}

// NewSpectrum will set up our spectrum
func NewSpectrum(rate float64, size int) *Spectrum {

	var s = &Spectrum{
		maxBins:    MaxBins,
		dataSize:   (size / 2) + 1,
		sampleSize: size,
		sampleRate: rate,
	}

	s.loCuts = make([]int, s.maxBins+1)
	s.hiCuts = make([]int, s.maxBins+1)

	s.Recalculate(s.maxBins, 20, s.sampleRate/2)

	return s
}

// DataSet reurns a new data set with settings matching this spectrum
func (s *Spectrum) DataSet(input []float64) *DataSet {

	slowMax := int((ScalingSlowWindow*s.sampleRate)/float64(s.sampleSize)) * 2
	fastMax := int((ScalingFastWindow*s.sampleRate)/float64(s.sampleSize)) * 2

	var set = &DataSet{
		id:         s.setCount,
		spectrum:   s,
		dataSize:   s.dataSize,
		dataBuf:    make([]complex128, s.dataSize),
		binBuf:     make([]float64, s.maxBins),
		prevBuf:    make([]float64, s.maxBins),
		slowWindow: NewMovingWindow(slowMax),
		fastWindow: NewMovingWindow(fastMax),
	}

	set.fftwPlan = fftw.New(input, set.dataBuf, s.sampleSize, fftw.Estimate)

	s.setCount++

	return set
}

// Recalculate rebuilds our frequency bins with bins bin counts
//
// reference: https://github.com/karlstav/cava/blob/master/cava.c#L654
// reference: https://github.com/noriah/cli-visualizer/blob/master/src/Transformer/SpectrumTransformer.cpp#L598
func (s *Spectrum) Recalculate(bins int, lo, hi float64) int {
	if bins > s.maxBins {
		bins = s.maxBins
	}

	s.numBins = bins

	var cBins = float64(bins + 1)

	var cFreq = math.Log10(lo/hi) / ((1 / cBins) - 1)

	// so this came from dpayne/cli-visualizer
	// until i can find a different solution
	for xBin := 0; xBin <= bins; xBin++ {
		// Fix issue where recalculations may not be accurate due to
		// previous runs
		s.loCuts[xBin] = 0
		s.hiCuts[xBin] = 0

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
func (s *Spectrum) Generate(dSet *DataSet) {
	dSet.numBins = s.numBins

	for xBin := 0; xBin <= dSet.numBins; xBin++ {

		var vM = 0.0

		for xF := s.loCuts[xBin]; xF <= s.hiCuts[xBin] &&
			xF > 0 &&
			xF < dSet.dataSize; xF++ {

			vM += pyt(dSet.dataBuf[xF])
		}

		vM /= float64(s.hiCuts[xBin] - s.loCuts[xBin] + 1)

		vM *= (math.Log2(float64(2+xBin)) * (100.0 / float64(dSet.numBins)))

		dSet.binBuf[xBin] = math.Pow(vM, 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}

// Scale scales the data
func Scale(height int, dSet *DataSet) {

	dSet.peakHeight = 0.125

	var vSilent = true

	for xBin := 0; xBin <= dSet.numBins; xBin++ {
		if dSet.binBuf[xBin] > 0 {
			vSilent = false
			if dSet.peakHeight < dSet.binBuf[xBin] {
				dSet.peakHeight = dSet.binBuf[xBin]
			}
		}
	}

	if vSilent {
		return
	}

	dSet.fastWindow.Update(dSet.peakHeight)

	var vMean, vSD = dSet.slowWindow.Update(dSet.peakHeight)

	if length := dSet.slowWindow.Len(); length > dSet.fastWindow.Cap() {
		var vMag = math.Abs(dSet.fastWindow.Mean() - vMean)
		if vMag > (ScalingResetDeviation * vSD) {
			dSet.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			vMean, vSD = dSet.slowWindow.Stats()
		}
	}

	var vMag = math.Max(vMean+(2*vSD), 1)

	for xBin, cHeight := 0, float64(height-1); xBin <= dSet.numBins; xBin++ {
		dSet.binBuf[xBin] = ((dSet.binBuf[xBin] / vMag) * (cHeight))

		dSet.binBuf[xBin] = math.Min(cHeight, dSet.binBuf[xBin])
	}
}

// Monstercat is not entirely understood yet.
// We need to work on it
func Monstercat(factor float64, dSet *DataSet) {

	// "pow is probably doing that same logarithm in every call, so you're
	//  extracting out half the work"
	var lf = math.Log(factor)

	for xBin := 0; xBin <= dSet.numBins; xBin++ {

		for xPass := 0; xPass <= dSet.numBins; xPass++ {

			var tmp = dSet.binBuf[xBin] / math.Exp(lf*absInt(xBin-xPass))

			if tmp > dSet.binBuf[xBin] {
				dSet.binBuf[xBin] = tmp
			}
		}
	}
}

func absInt(value int) float64 {
	return math.Abs(float64(value))
}

// Falloff does falling off things
func Falloff(weight float64, dSet *DataSet) {
	weight = math.Max(0.7, math.Min(1, weight))

	for xBin := 0; xBin <= dSet.numBins; xBin++ {
		mag := falloff(weight, dSet.prevBuf[xBin], dSet.binBuf[xBin])
		dSet.binBuf[xBin] = mag
		dSet.prevBuf[xBin] = mag
	}
}

func falloff(weight, prev, now float64) float64 {
	return math.Max(math.Min(prev-1, prev*weight), now)
}
