package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"

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
	// PeakThreshold is the threshold to not draw if the peak is less.
	PeakThreshold = 0.01
)

type processor struct {
	cfg *Config

	slowWindow util.MovingWindow
	fastWindow util.MovingWindow

	fftBuf    []complex128
	inputBufs [][]input.Sample
	barBufs   [][]input.Sample

	plans    []*fft.Plan
	spectrum dsp.Spectrum

	bars    int
	display graphic.Display
}

// Catnip starts to draw the visualizer on the termbox screen.
func Catnip(cfg *Config) error {
	// allocate as much as possible as soon as possible
	var (
		barBufFull = make([]float64, cfg.ChannelCount*cfg.SampleSize)

		slowMax    = ((int(ScalingSlowWindow * cfg.SampleRate)) / cfg.SampleSize) * 2
		fastMax    = ((int(ScalingFastWindow * cfg.SampleRate)) / cfg.SampleSize) * 2
		windowData = make([]float64, slowMax+fastMax)
	)

	var sessConfig = input.SessionConfig{
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	}

	proc := processor{
		cfg: cfg,
		slowWindow: util.MovingWindow{
			Data:     windowData[0:slowMax],
			Capacity: slowMax,
		},
		fastWindow: util.MovingWindow{
			Data:     windowData[slowMax : slowMax+fastMax],
			Capacity: fastMax,
		},

		fftBuf:    make([]complex128, cfg.fftSize),
		inputBufs: input.MakeBuffers(sessConfig),
		barBufs:   make([][]float64, cfg.ChannelCount),

		plans: make([]*fft.Plan, cfg.ChannelCount),
		spectrum: dsp.Spectrum{
			SampleRate: cfg.SampleRate,
			SampleSize: cfg.SampleSize,
			Bins:       make(dsp.BinBuf, cfg.SampleSize),
		},

		bars:    0,
		display: graphic.Display{},
	}

	for idx := range proc.barBufs {
		proc.barBufs[idx] = barBufFull[(idx * cfg.SampleSize):((idx + 1) * cfg.SampleSize)]
	}

	for idx, buf := range proc.inputBufs {
		proc.plans[idx] = &fft.Plan{
			Input:  buf,
			Output: proc.fftBuf,
		}
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

	proc.spectrum.SetSmoothing(cfg.SmoothFactor)
	proc.spectrum.SetWinVar(cfg.WinVar)
	proc.spectrum.SetType(dsp.SpectrumType(cfg.SpectrumType))

	proc.display.SetWidths(cfg.BarWidth, cfg.SpaceWidth)
	proc.display.SetBase(cfg.BaseThick)
	proc.display.SetDrawType(graphic.DrawType(cfg.DrawType))
	proc.display.SetStyles(cfg.Styles)

	if err = proc.display.Init(); err != nil {
		return err
	}
	defer proc.display.Close()

	endSig := make(chan os.Signal, 2)
	signal.Notify(endSig, os.Interrupt)
	signal.Notify(endSig, os.Kill)

	// Root Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the display
	// replace our context so display can signal quit
	ctx = proc.display.Start(ctx)
	defer proc.display.Stop()

	if err := audio.Start(ctx, proc.inputBufs, &proc); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}

	return nil
}

// Catnip starts to draw the visualizer on the termbox screen.
func (proc *processor) Process() {
	if n := proc.display.Bars(proc.cfg.ChannelCount); n != proc.bars {
		proc.bars = proc.spectrum.Recalculate(n)
	}

	var peak float64

	for idx, buf := range proc.barBufs {
		window.CosSum(proc.inputBufs[idx], proc.cfg.WinVar)
		proc.plans[idx].Execute()
		proc.spectrum.Process(buf, proc.fftBuf)

		for _, v := range buf[:proc.bars] {
			if peak < v {
				peak = v
			}
		}
	}

	// Don't draw if the peak is too small to even draw.
	if peak <= PeakThreshold {
		return
	}

	var scale = 1.0

	// do some scaling if we are above 0
	if peak > 0.0 {
		proc.fastWindow.Update(peak)
		vMean, vSD := proc.slowWindow.Update(peak)

		if length := proc.slowWindow.Len(); length >= proc.fastWindow.Cap() {
			if math.Abs(proc.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
				vMean, vSD = proc.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			}
		}

		if t := vMean + (1.5 * vSD); t > 1.0 {
			scale = t
		}
	}

	proc.display.Draw(proc.barBufs, proc.cfg.ChannelCount, proc.bars, scale)
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
