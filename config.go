package main

import (
	"github.com/noriah/tavis/graphic"
	"github.com/noriah/tavis/input"
)

// Config is a temporary struct to define parameters
type Config struct {
	// InputBackend is the backend that the input belongs to
	InputBackend input.Backend
	// InputDevice is the device we want to listen to
	InputDevice input.Device
	// SampleRate is the rate at which samples are read
	SampleRate float64
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// SmoothFactor factor of smooth
	SmoothFactor float64
	// Gamma factor of distribution
	Gamma float64
	// BaseThick number of cells wide/high the base is
	BaseThick int
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// SampleSize is how much we draw. Play with it
	SampleSize int
	// DrawType is the draw type
	DrawType int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
}

// NewZeroConfig returns a zero config
func NewZeroConfig() Config {
	return Config{
		SampleRate:   44100,
		SmoothFactor: 52.5,
		Gamma:        2.0,
		BaseThick:    1,
		BarWidth:     2,
		SpaceWidth:   1,
		SampleSize:   1024,
		DrawType:     int(graphic.DrawDefault),
		ChannelCount: 2,
	}
}

func sanitizeConfig(cfg *Config) error {

	switch {
	case cfg.SmoothFactor > 100.0:
		cfg.SmoothFactor = 1.0
	case cfg.SmoothFactor <= 0.0:
		cfg.SmoothFactor = 0.00001
	default:
		cfg.SmoothFactor /= 100.0
	}

	return nil
}
