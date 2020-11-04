package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
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
	ScalingDumpPercent = 0.75
	// ScalingResetDeviation standard deviations from the mean before reset
	ScalingResetDeviation = 1.0
)

// Catnip starts to draw the visualizer on the termbox screen.
func Catnip(cfg *Config) error {
	// allocate as much as possible as soon as possible
	var (
		barBufFull = make([]float64, cfg.ChannelCount*cfg.SampleSize)
		fftBuf     = make([]complex128, cfg.fftSize)
		spBinBuf   = make(dsp.BinBuf, cfg.SampleSize)

		slowMax    = ((int(ScalingSlowWindow * cfg.SampleRate)) / cfg.SampleSize) * 2
		fastMax    = ((int(ScalingFastWindow * cfg.SampleRate)) / cfg.SampleSize) * 2
		windowData = make([]float64, slowMax+fastMax)

		barBufs = make([][]float64, cfg.ChannelCount)
		plans   = make([]*fft.Plan, cfg.ChannelCount)
	)

	var slowWindow = &util.MovingWindow{
		Data:     windowData[0:slowMax],
		Capacity: slowMax,
	}

	var fastWindow = &util.MovingWindow{
		Data:     windowData[slowMax : slowMax+fastMax],
		Capacity: fastMax,
	}

	for idx := range barBufs {
		barBufs[idx] = barBufFull[(idx * cfg.SampleSize):((idx + 1) * cfg.SampleSize)]
	}

	// DrawDelay is the time we wait between ticks to draw.
	var drawDelay = time.Second / time.Duration(
		int((cfg.SampleRate / float64(cfg.SampleSize))))

	// INPUT SETUP

	var backend, err = initBackend(cfg)
	if err != nil {
		return err
	}

	var sessConfig = input.SessionConfig{
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	}

	if sessConfig.Device, err = getDevice(backend, cfg); err != nil {
		return err
	}

	audio, err := backend.Start(sessConfig)
	defer backend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	// Make a spectrum
	var spectrum = dsp.Spectrum{
		SampleRate: cfg.SampleRate,
		SampleSize: cfg.SampleSize,
		Bins:       spBinBuf,
	}

	spectrum.SetSmoothing(cfg.SmoothFactor)
	spectrum.SetWinVar(cfg.WinVar)
	spectrum.SetType(dsp.SpectrumType(cfg.SpectrumType))

	var display = graphic.Display{}
	defer display.Close()

	if err = display.Init(); err != nil {
		return err
	}

	display.SetWidths(cfg.BarWidth, cfg.SpaceWidth)
	display.SetBase(cfg.BaseThick)
	display.SetDrawType(graphic.DrawType(cfg.DrawType))
	display.SetStyles(cfg.Styles)

	var endSig = make(chan os.Signal, 2)
	signal.Notify(endSig, os.Interrupt)
	signal.Notify(endSig, os.Kill)

	// Root Context
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var inputBufs = audio.SampleBuffers()

	for idx, buf := range inputBufs {
		plans[idx] = &fft.Plan{
			Input:  buf,
			Output: fftBuf,
		}
	}

	// Start the display
	// replace our context so display can signal quit
	ctx = display.Start(ctx)
	defer display.Stop()

	if err := audio.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}

	defer audio.Stop()

	var barCount int

	var timer = time.NewTimer(drawDelay)

	defer func(t *time.Timer) {
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
	}(timer)

	for {
		if audio.ReadyRead() < cfg.SampleSize {
			continue
		}

		if err = audio.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		if barVar := display.Bars(cfg.ChannelCount); barVar != barCount {
			barCount = spectrum.Recalculate(barVar)
		}

		var peak = 0.0

		for idx, buf := range barBufs {
			window.CosSum(inputBufs[idx], cfg.WinVar)
			plans[idx].Execute()
			spectrum.Process(buf, fftBuf)

			for _, v := range buf[:barCount] {
				if peak < v {
					peak = v
				}
			}
		}

		var scale = 1.0

		// do some scaling if we are above 0
		if peak > 0.0 {
			fastWindow.Update(peak)
			var vMean, vSD = slowWindow.Update(peak)

			if length := slowWindow.Len(); length >= fastWindow.Cap() {

				if math.Abs(fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
					vMean, vSD = slowWindow.Drop(
						int(float64(length) * ScalingDumpPercent))
				}
			}

			if t := vMean + (1.5 * vSD); t > 1.0 {
				scale = t
			}
		}

		display.Draw(barBufs, cfg.ChannelCount, barCount, scale)

		select {
		case <-endSig:
			return nil
		case <-ctx.Done():
			return nil
		case <-timer.C:
			timer.Reset(drawDelay)
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
