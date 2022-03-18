package catnip

import (
	"context"
	"fmt"
	"time"

	"github.com/noriah/catnip/config"
	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/visualizer"

	"github.com/pkg/errors"
)

// Catnip starts to draw the visualizer on the termbox screen.
func Catnip(cfg *config.Config) error {
	// allocate as much as possible as soon as possible

	var sessConfig = input.SessionConfig{
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	}

	display := &graphic.Display{}

	inputBuffers := input.MakeBuffers(cfg.ChannelCount, cfg.SampleSize)
	vis := visualizer.New(cfg, inputBuffers)

	vis.Spectrum = dsp.NewSpectrum(cfg)
	vis.Display = display

	// INPUT SETUP

	var backend, err = InitBackend(cfg)
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

	if err = display.Init(); err != nil {
		return err
	}
	defer display.Close()

	display.SetSizes(cfg.BarSize, cfg.SpaceSize)
	display.SetBase(cfg.BaseSize)
	display.SetDrawType(graphic.DrawType(cfg.DrawType))
	display.SetStyles(cfg.Styles)

	// Root Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the display
	// replace our context so display can signal quit
	ctx = display.Start(ctx)
	defer display.Stop()

	if cfg.FrameRate > 0 {
		go func() {
			ticker := time.NewTicker(time.Second / time.Duration(cfg.FrameRate))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					vis.Draw(true)
				}
			}
		}()
	}

	if err := audio.Start(ctx, inputBuffers, vis); err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			return errors.Wrap(err, "failed to start input session")
		}
	}

	return nil
}

func InitBackend(cfg *config.Config) (input.Backend, error) {
	var backend = input.FindBackend(cfg.Backend)
	if backend == nil {
		return nil, fmt.Errorf("backend not found: %q; check list-backends", cfg.Backend)
	}

	if err := backend.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize input backend")
	}

	return backend, nil
}

func getDevice(backend input.Backend, cfg *config.Config) (input.Device, error) {
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
