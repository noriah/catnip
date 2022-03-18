package config

import (
	"errors"

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
	// FrameRate is the number of frames to draw every second (0 draws it every
	// perfect sample)
	FrameRate int
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// SmoothFactor factor of smooth
	SmoothFactor float64
	// WinVar factor of distribution
	WinVar float64
	// BaseSize number of cells wide/high the base is
	BaseSize int
	// BarSize is the size of bars, in columns/rows
	BarSize int
	// SpaceSize is the size of spaces, in columns/rows
	SpaceSize int
	// SampleSiz is how much we draw. Play with it
	SampleSize int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
	// Combine determines if we merge streams (stereo -> mono)
	Combine bool
	// DrawType is the draw type
	DrawType int
	// Styles is the configuration for bar color styles
	Styles graphic.Styles
}

// NewZeroConfig returns a zero config
// it is the "default"
//
// nora's defaults:
//  - sampleRate: 122880
//  - sampleSize: 2048
//  - smoothFactor: 80.15
//  - super smooth detail view
func NewZeroConfig() Config {
	return Config{
		Backend:      "portaudio",
		SampleRate:   44100,
		FrameRate:    0,
		SmoothFactor: 80.15,
		WinVar:       0.50, // Deprecated
		BaseSize:     1,
		BarSize:      2,
		SpaceSize:    1,
		SampleSize:   1024,
		ChannelCount: 2,
		Combine:      false,
		DrawType:     int(graphic.DrawDefault),
	}
}

// Sanitize cleans things up
func (cfg *Config) Sanitize() error {

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
