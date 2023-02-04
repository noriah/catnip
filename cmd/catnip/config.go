package main

import (
	"errors"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
)

// Config is a temporary struct to define parameters
type config struct {
	// Backend is the backend name from list-backends
	backend string
	// Device is the device name from list-devices
	device string
	// SampleRate is the rate at which samples are read
	sampleRate float64
	// SmoothFactor factor of smooth
	smoothFactor float64
	// Smoothing method used to do time smoothing.
	smoothingMethod int
	// Size of window used for averaging methods.
	smoothingAverageWindowSize int
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
	// DrawType is the draw type
	drawType int
	// Combine determines if we merge streams (stereo -> mono)
	combine bool
	// Don't run math.Log on the output of the analyzer
	dontNormalize bool
	// Use threaded processor
	useThreaded bool
	// Invert the order of bin drawing
	invertDraw bool
	// Styles is the configuration for bar color styles
	styles graphic.Styles
}

// NewZeroConfig returns a zero config
// it is the "default"
func newZeroConfig() config {
	return config{
		backend:                    input.DefaultBackend(),
		sampleRate:                 44100,
		sampleSize:                 1024,
		smoothFactor:               64.15,
		smoothingMethod:            int(dsp.SmoothDefault),
		smoothingAverageWindowSize: 0, // if zero, will be calculated
		frameRate:                  0,
		baseSize:                   1,
		barSize:                    1,
		spaceSize:                  1,
		channelCount:               2,
		drawType:                   int(graphic.DrawDefault),
		dontNormalize:              false,
		combine:                    false,
		useThreaded:                false,
		invertDraw:                 false,
	}
}

// Sanitize cleans things up
func (cfg *config) validate() error {

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
