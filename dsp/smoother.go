package dsp

import (
	"math"

	"github.com/noriah/catnip/util"
)

type SmoothingMethod int

const (
	SmoothUnspecified   SmoothingMethod = iota // 0
	SmoothSimple                               // 1
	SmoothAverage                              // 2
	SmoothSimpleAverage                        // 3
	SmoothNew                                  // 5
	SmoothNewAverage                           // 4

	SmoothDefault = SmoothSimpleAverage
)

type SmootherConfig struct {
	AverageSize     int             // size of window for average methods
	ChannelCount    int             // number of channels
	SampleSize      int             // number of samples per slice
	SampleRate      float64         // sample rate
	SmoothingFactor float64         // smoothing factor
	SmoothingMethod SmoothingMethod // smoothing method
}

type Smoother interface {
	SmoothBuffers([][]float64)
	SmoothBin(int, int, float64) float64
}

type smoother struct {
	values       [][]float64 // old values used for smoothing
	averages     [][]*util.MovingWindow
	smoothFactor float64 // smothing factor
	smoothMethod SmoothingMethod
}

func NewSmoother(cfg SmootherConfig) Smoother {
	if cfg.SmoothingFactor <= 0.0 {
		cfg.SmoothingFactor = math.SmallestNonzeroFloat64
	}

	sm := &smoother{
		values:       make([][]float64, cfg.ChannelCount),
		averages:     make([][]*util.MovingWindow, cfg.ChannelCount),
		smoothFactor: cfg.SmoothingFactor,
		smoothMethod: cfg.SmoothingMethod,
	}

	// calculate the window size if its not set
	size := cfg.AverageSize
	if size < 1 {
		rate := cfg.SampleRate / float64(cfg.SampleSize)
		// this is based on my experimentation. no evidence that it is best.
		// the goal of the window average is to smooth out the very minor amplitude
		// changes. the visual representation of them is a function of the frame
		// rate. assuming a base of 60FPS, this should handle it pretty well for
		// most rate/size combinations.
		size = int(math.Ceil(5.0 * (rate / 60.0)))
	}

	for idx := range sm.values {
		sm.values[idx] = make([]float64, cfg.SampleSize)
		sm.averages[idx] = make([]*util.MovingWindow, cfg.SampleSize)
		for i := range sm.averages[idx] {
			sm.averages[idx][i] = util.NewMovingWindow(size)
		}
	}

	return sm
}

func (sm *smoother) SmoothBuffers(bufs [][]float64) {
	peak := 0.0
	for _, buf := range bufs {
		for _, v := range buf {
			if v > peak {
				peak = v
			}
		}
	}

	for ch, buf := range bufs {
		for idx, v := range buf {
			buf[idx] = sm.switchSmoothing(ch, idx, v, peak)
		}
	}
}

func (sm *smoother) SmoothBin(ch, idx int, value float64) float64 {
	return sm.switchSmoothing(ch, idx, value, 0.0)
}

func (sm *smoother) switchSmoothing(ch, idx int, value, peak float64) float64 {
	switch sm.smoothMethod {
	default:
	case SmoothUnspecified:
	case SmoothSimple:
		return sm.smoothBinSimple(ch, idx, value)
	case SmoothAverage:
		return sm.smoothBinAverage(ch, idx, value)
	case SmoothSimpleAverage:
		v := sm.smoothBinAverage(ch, idx, value)
		return sm.smoothBinSimple(ch, idx, v)
	case SmoothNew:
		return sm.smoothBinNew(ch, idx, value, peak)
	case SmoothNewAverage:
		v := sm.smoothBinAverage(ch, idx, value)
		return sm.smoothBinNew(ch, idx, v, peak)
	}

	return 0.0
}

func (sm *smoother) smoothBinSimple(ch, idx int, value float64) float64 {
	value *= 1.0 - sm.smoothFactor
	value += sm.values[ch][idx] * sm.smoothFactor
	sm.values[ch][idx] = value
	return value
}

func (sm *smoother) smoothBinAverage(ch, idx int, value float64) float64 {
	if math.IsNaN(value) {
		value = 0.0
	}
	avg, _ := sm.averages[ch][idx].Update(value)
	return avg
}

func (sm *smoother) smoothBinNew(ch, idx int, value, peak float64) float64 {
	if math.IsNaN(value) {
		value = 0.0
	}

	existing := sm.values[ch][idx]

	if math.IsNaN(existing) {
		existing = 0.0
	}

	diff := math.Abs(value - existing)
	max := math.Max(value, existing)

	diffPct := diff / max
	peakValuePct := value / math.Max(1.0, peak)

	factor := sm.smoothFactor
	partial := (1.0 - factor) * 0.45

	factor += partial - ((partial + 0.1) * math.Pow(diffPct, 1.5))
	factor += (partial / 0.75) - ((partial / 0.5) * math.Pow(peakValuePct, 4.0))

	// Clamp the factor between 0+MinFloat64, and 1-MinFloat64 so that it does
	// not zero out the value or not change at all.
	factor = math.Max(
		math.SmallestNonzeroFloat64,
		math.Min(
			1.0-math.SmallestNonzeroFloat64,
			factor))

	value *= 1.0 - factor
	value += existing * factor

	sm.values[ch][idx] = value

	return value
}
