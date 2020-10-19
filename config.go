package main

import "github.com/noriah/tavis/input"

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
	// SmoothResponse response value
	SmoothResponse float64
	// BaseThick number of cells wide/high the base is
	BaseThick int
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
	// MaxBins maximum number of bins
	MaxBins int
}

// NewZeroConfig returns a zero config
func NewZeroConfig() Config {
	return Config{
		SampleRate:     44100,
		LoCutFreq:      20,
		HiCutFreq:      22050,
		SmoothFactor:   52.5,
		SmoothResponse: 43.5,
		BaseThick:      1,
		BarWidth:       2,
		SpaceWidth:     1,
		TargetFPS:      60,
		ChannelCount:   2,
		MaxBins:        256,
	}
}
