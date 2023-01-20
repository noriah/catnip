package dsp

import "math"

type SmootherConfig struct {
	SampleSize      int     // number of samples per slice
	ChannelCount    int     // number of channels
	SmoothingFactor float64 // smoothing factor
	NewSmoothing    bool    // use new smoothing method
}

type Smoother interface {
	SmoothBuffers([][]float64)
	SmoothBin(int, int, float64, float64) float64
}

type smoother struct {
	values       [][]float64 // old values used for smoothing
	smoothFactor float64     // smothing factor
	newSmoothing bool        // use new smoothing method
}

func NewSmoother(cfg SmootherConfig) Smoother {
	sm := &smoother{
		values:       make([][]float64, cfg.ChannelCount),
		newSmoothing: cfg.NewSmoothing,
	}

	for idx := range sm.values {
		sm.values[idx] = make([]float64, cfg.SampleSize)
	}

	sm.setSmoothing(cfg.SmoothingFactor)

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
			buf[idx] = sm.SmoothBin(ch, idx, v, peak)
		}
	}
}

func (sm *smoother) SmoothBin(ch, idx int, value, peak float64) float64 {
	existing := sm.values[ch][idx]
	factor := sm.smoothFactor

	if sm.newSmoothing {

		if math.IsNaN(value) {
			value = 0.0
		}

		if math.IsNaN(existing) {
			existing = 0.0
		}

		diff := math.Abs(value - existing)
		max := math.Max(value, existing)
		// average := (value + existing) / 2.0

		diffPct := diff / max
		peakDiffPct := value / math.Max(1, peak)

		partial := (1.0 - factor) * 0.45
		factor += partial - ((partial + 0.01) * math.Pow(diffPct, 1.5))
		factor += (partial / 2.0) - (((partial / 2.0) + 0.00) * math.Pow(peakDiffPct, 4.0))

		factor = math.Max(
			math.SmallestNonzeroFloat64,
			math.Min(
				1.0-math.SmallestNonzeroFloat64,
				factor))
	}

	value *= 1.0 - factor
	value += existing * factor

	sm.values[ch][idx] = value

	return value
}

// SetSmoothing sets the smoothing parameters
func (sm *smoother) setSmoothing(factor float64) {
	if factor <= 0.0 {
		factor = math.SmallestNonzeroFloat64
	}

	sf := math.Pow(10.0, (1.0-factor)*(-25.0))

	// roughly 2048/122800
	sm.smoothFactor = math.Pow(sf, 0.0167)
}
