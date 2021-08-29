package main

import (
	"context"
	"fmt"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/fft"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/util"

	"github.com/pkg/errors"
)

const (
	// ScalingSlowWindow in seconds
	ScalingSlowWindow = 5
	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2
	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.60
	// ScalingResetDeviation standard deviations from the mean before reset
	ScalingResetDeviation = 1.0
	// PeakThreshold is the threshold to not draw if the peak is less.
	PeakThreshold = 0.01
)

// Catnip starts to draw the visualizer on the termbox screen.
func Catnip(cfg *Config) error {
	// allocate as much as possible as soon as possible
	var (

		// slowMax/fastMax
		slowMax = ((int(ScalingSlowWindow * cfg.SampleRate)) / cfg.SampleSize) * 2
		fastMax = ((int(ScalingFastWindow * cfg.SampleRate)) / cfg.SampleSize) * 2

		total = ((cfg.ChannelCount * cfg.SampleSize) * 2) + (slowMax + fastMax)

		floatData = make([]float64, total)
	)

	var sessConfig = input.SessionConfig{
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	}

	vis := visualizer{
		cfg: cfg,
		slowWindow: util.MovingWindow{
			Capacity: slowMax,
			Data:     floatData[:slowMax],
		},
		fastWindow: util.MovingWindow{
			Capacity: fastMax,
			Data:     floatData[slowMax : slowMax+fastMax],
		},

		fftBuf:    make([]complex128, cfg.fftSize),
		inputBufs: make([][]float64, cfg.ChannelCount),
		barBufs:   make([][]float64, cfg.ChannelCount),

		plans: make([]*fft.Plan, cfg.ChannelCount),
		spectrum: dsp.Spectrum{
			SampleRate: cfg.SampleRate,
			SampleSize: cfg.SampleSize,
			Bins:       make([]dsp.Bin, cfg.SampleSize),
			OldValues:  make([][]float64, cfg.ChannelCount),
		},

		bars:    0,
		display: graphic.Display{},
	}

	var pos = slowMax + fastMax
	for idx := range vis.barBufs {

		vis.barBufs[idx] = floatData[pos : pos+cfg.SampleSize]
		pos += cfg.SampleSize

		vis.inputBufs[idx] = floatData[pos : pos+cfg.SampleSize]
		pos += cfg.SampleSize

		vis.plans[idx] = &fft.Plan{
			Input:  vis.inputBufs[idx],
			Output: vis.fftBuf,
		}

		vis.spectrum.OldValues[idx] = make([]float64, cfg.SampleSize)

		vis.plans[idx].Init()
	}

	// INPUT SETUP

	var backend, err = initBackend(cfg)
	if err != nil {
		return err
	}

	if sessConfig.Device, err = getDevice(backend, cfg); err != nil {
		return err
	}

	audio, err := backend.Start(sessConfig)
	defer backend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	vis.spectrum.SetSmoothing(cfg.SmoothFactor)
	vis.spectrum.SetWinVar(cfg.WinVar)

	if err = vis.display.Init(); err != nil {
		return err
	}
	defer vis.display.Close()

	vis.display.SetSizes(cfg.BarSize, cfg.SpaceSize)
	vis.display.SetBase(cfg.BaseSize)
	vis.display.SetDrawType(graphic.DrawType(cfg.DrawType))
	vis.display.SetStyles(cfg.Styles)

	// Root Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the display
	// replace our context so display can signal quit
	ctx = vis.display.Start(ctx)
	defer vis.display.Stop()

	if err := audio.Start(ctx, vis.inputBufs, &vis); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}

	return nil
}

func initBackend(cfg *Config) (input.Backend, error) {
	var backend = input.FindBackend(cfg.Backend)
	if backend == nil {
		return nil, fmt.Errorf("backend not found: %q; check list-backends", cfg.Backend)
	}

	if err := backend.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize input backend")
	}

	return backend, nil
}

func getDevice(backend input.Backend, cfg *Config) (input.Device, error) {
	if cfg.Device == "" {
		var def, err = backend.DefaultDevice()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get default device")
		}
		return def, nil
	}

	var devices, err = backend.Devices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get devices")
	}

	for idx := range devices {
		if devices[idx].String() == cfg.Device {
			return devices[idx], nil
		}
	}

	return nil, errors.Errorf("device %q not found; check list-devices", cfg.Device)
}
