package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/dsp"
	"github.com/noriah/tavis/dsp/window"
	"github.com/noriah/tavis/graphic"
	"github.com/noriah/tavis/input"

	"github.com/pkg/errors"
)

// Run starts to draw the visualizer on the tcell Screen.
func Run(cfg Config) error {
	// DrawDelay is the time we wait between ticks to draw.
	var drawDelay = time.Second / time.Duration(
		int((cfg.SampleRate / float64(cfg.SampleSize))))

	var audio, err = cfg.InputBackend.Start(input.SessionConfig{
		Device:     cfg.InputDevice,
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	})
	defer cfg.InputBackend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(cfg.SampleRate, cfg.SampleSize)
	spectrum.SetSmoothing(cfg.SmoothFactor)
	spectrum.SetWinVar(cfg.WinVar)
	spectrum.SetType(dsp.SpectrumType(cfg.SpectrumType))

	for ch := 0; ch < cfg.ChannelCount; ch++ {
		spectrum.AddStream(audio.SampleBuffers()[ch])
	}

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
