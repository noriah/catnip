package main

import (
	"errors"

	"github.com/noriah/catnip/graphic"
)

// Config is a temporary struct to define parameters
type config struct {
	// Backend is the backend name from list-backends
	backend string
	// Device is the device name from list-devices
	device string
	// SampleRate is the rate at which samples are read
	sampleRate float64
	// LoCutFrqq is the low end of our audio spectrum
	// Deprecated
	// loCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	// Deprecated
	// hiCutFreq float64
	// SmoothFactor factor of smooth
	smoothFactor float64
	// SampleSize is how much we draw. Play with it
	sampleSize int
	// FrameRate is the number of frames to draw every second (0 draws it every
	// perfect sample)
	frameRate int
	// BaseSize number of cells wide/high the base is
	baseSize int
	// BarSize is the size of bars, in columns/rows
	barSize int
	// SpaceSize is the size of spaces, in columns/rows
	spaceSize int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	channelCount int
	// Combine determines if we merge streams (stereo -> mono)
	drawType int
	// Styles is the configuration for bar color styles
	combine bool
	// Use threaded processor
	useThreaded bool
	// Invert the order of bin drawing
	invertDraw bool
	// DrawType is the draw type
	styles graphic.Styles
}

// NewZeroConfig returns a zero config
// it is the "default"
//
// nori's defaults:
//   - sampleRate: 122880
//   - sampleSize: 2048
//   - smoothFactor: 80.15
//   - super smooth detail view
func newZeroConfig() config {
	return config{
		sampleRate:   44100,
		sampleSize:   1024,
		smoothFactor: 80.15,
		frameRate:    0,
		baseSize:     1,
		barSize:      2,
		spaceSize:    1,
		channelCount: 2,
		drawType:     int(graphic.DrawDefault),
		combine:      false,
		useThreaded:  false,
	}
}

// Sanitize cleans things up
func (cfg *config) Sanitize() error {

	if cfg.sampleRate < float64(cfg.sampleSize) {
		return errors.New("sample rate lower than sample size")
	}

	if cfg.sampleSize < 4 {
		return errors.New("sample size too small (4+ required)")
	}

	switch {

	case cfg.channelCount > 2:
		return errors.New("too many channels (2 max)")

	case cfg.channelCount < 1:
		return errors.New("too few channels (1 min)")

	}

	switch {
	case cfg.smoothFactor > 99.99:
		cfg.smoothFactor = 0.9999
	case cfg.smoothFactor < 0.00001:
		cfg.smoothFactor = 0.00001
	default:
		cfg.smoothFactor /= 100.0
	}

	return nil
}
