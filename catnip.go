package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"

	"github.com/pkg/errors"
)

// Catnip starts to draw the visualizer on the termbox screen.
func Catnip(cfg *Config) error {

	// DrawDelay is the time we wait between ticks to draw.
	var drawDelay = time.Second / time.Duration(
		int((cfg.SampleRate / float64(cfg.SampleSize))))

	var backend, err = initBackend(cfg)
	if err != nil {
		return err
	}

	device, err := getDevice(backend, cfg)

	if err != nil {
		return err
	}

	audio, err := backend.Start(input.SessionConfig{
		Device:     device,
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	})
	defer backend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(cfg.SampleRate, cfg.SampleSize)
	spectrum.SetSmoothing(cfg.SmoothFactor)
	spectrum.SetWinVar(cfg.WinVar)
	spectrum.SetType(dsp.SpectrumType(cfg.SpectrumType))
	spectrum.AddStream(audio.SampleBuffers()...)

	var barBuffers = spectrum.BinBuffers()

	var display = graphic.NewDisplay(cfg.SampleRate, cfg.SampleSize)
	defer display.Close()

	if err = display.Init(); err != nil {
		return err
	}

	display.SetWidths(cfg.BarWidth, cfg.SpaceWidth)
	display.SetBase(cfg.BaseThick)
	display.SetDrawType(graphic.DrawType(cfg.DrawType))

	var endSig = make(chan os.Signal, 2)
	signal.Notify(endSig, os.Interrupt)
	signal.Notify(endSig, os.Kill)

	// Root Context
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Start the display
	// replace our context so display can signal quit
	ctx = display.Start(ctx)
	defer display.Stop()

	var barCount int

	// Make a window function for use with spectrum
	var win = func(buf []float64) {
		window.CosSum(buf, cfg.WinVar)
	}

	if err := audio.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer audio.Stop()

	var timer = time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-endSig:
			return nil
		case <-ctx.Done():
			return nil
		case <-timer.C:
			timer.Reset(drawDelay)
		}

		if audio.ReadyRead() < cfg.SampleSize {
			continue
		}

		if err = audio.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		if termWidth := display.Bars(cfg.ChannelCount); barCount != termWidth {
			barCount = spectrum.Recalculate(termWidth)
		}

		spectrum.Process(win)

		if err = display.Draw(barBuffers, cfg.ChannelCount, barCount); err != nil {
			return errors.Wrap(err, "graphic threw error. this should not happen.")
		}
	}
}

func initBackend(cfg *Config) (input.Backend, error) {

	var backend = input.FindBackend(cfg.Backend)
	if backend == nil {
		return nil, fmt.Errorf("backend not found: %q", cfg.Backend)
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
