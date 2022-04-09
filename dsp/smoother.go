package dsp

import "math"

type SmootherConfig struct {
	SampleSize      int     // number of samples per slice
	ChannelCount    int     // number of channels
	SmoothingFactor float64 // smoothing factor
}

type Smoother interface {
	SmoothBin(int, int, float64) float64
}

type smoother struct {
	values       [][]float64 // old values used for smoothing
	smoothFactor float64     // smothing factor
}

func NewSmoother(cfg SmootherConfig) Smoother {
	sm := &smoother{
		values: make([][]float64, cfg.ChannelCount),
	}

	for idx := range sm.values {
		sm.values[idx] = make([]float64, cfg.SampleSize)
	}

	sm.setSmoothing(cfg.SmoothingFactor)

	return sm
}

func (sm *smoother) SmoothBin(ch, idx int, value float64) float64 {
	value *= 1.0 - sm.smoothFactor
	value += sm.values[ch][idx] * sm.smoothFactor

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
