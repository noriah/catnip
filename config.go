package main

import (
	"errors"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/graphic"
)

// Config is a temporary struct to define parameters
type Config struct {
	// Backend is the backend name from list-backends
	Backend string
	// Device is the device name from list-devices
	Device string
	// SampleRate is the rate at which samples are read
	SampleRate float64
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// SmoothFactor factor of smooth
	SmoothFactor float64
	// WinVar factor of distribution
	WinVar float64
	// BaseThick number of cells wide/high the base is
	BaseThick int
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// SampleSiz is how much we draw. Play with it
	SampleSize int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
	// DrawType is the draw type
	DrawType int
	// SpectrumType is the spectrum calculation method
	SpectrumType int
}

// NewZeroConfig returns a zero config
// it is the "default"
//
// nora's defaults:
//  - sampleRate: 122880
//  - sampleSize: 2048
//  - super smooth detail view
func NewZeroConfig() *Config {
	return &Config{
		Backend:      "portaudio",
		SampleRate:   44100,
		SmoothFactor: 40.69,
		WinVar:       0.50,
		BaseThick:    1,
		BarWidth:     2,
		SpaceWidth:   1,
		SampleSize:   1024,
		ChannelCount: 2,
		DrawType:     int(graphic.DrawDefault),
		SpectrumType: int(dsp.SpectrumDefault),
	}
}

func sanitizeConfig(cfg *Config) error {

	if cfg.SampleRate < float64(cfg.SampleSize) {
		return errors.New("sample rate lower than sample size")
	}

	if cfg.SampleSize < 4 {
		return errors.New("sample size too small (4+ required)")
	}

	switch {

	case cfg.ChannelCount > 2:
		return errors.New("too many channels (2 max)")

	case cfg.ChannelCount < 1:
		return errors.New("too few channels (1 min)")

	}

	switch {
	case cfg.WinVar > 1.0:
		cfg.WinVar = 1.0
	case cfg.WinVar < 0.0:
		cfg.WinVar = 0.0
	default:
	}

	switch {
	case cfg.SmoothFactor > 99.99:
		cfg.SmoothFactor = 0.9999
	case cfg.SmoothFactor < 0.00001:
		cfg.SmoothFactor = 0.00001
	default:
		cfg.SmoothFactor /= 100.0
	}

	return nil
}
