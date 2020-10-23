package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/dsp"
	"github.com/noriah/tavis/graphic"
	"github.com/noriah/tavis/input"

	"github.com/pkg/errors"
)

// Run starts to draw the visualizer on the tcell Screen.
func Run(cfg Config) error {
	if err := sanitizeConfig(&cfg); err != nil {
		return err
	}

	var workRate = int(cfg.SampleRate / float64(cfg.SampleSize))

	// DrawDelay is the time we wait between ticks to draw.
	var drawDelay = time.Second / time.Duration(workRate+1)

	// Draw type
	var drawType = graphic.DrawType(cfg.DrawType)

	var audio, err = cfg.InputBackend.Start(input.SessionConfig{
		Device:     cfg.InputDevice,
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	})

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	var display = graphic.NewDisplay(cfg.SampleRate, cfg.SampleSize)
	defer display.Close()

	display.SetWidths(cfg.BarWidth, cfg.SpaceWidth)
	display.SetBase(cfg.BaseThick)
	display.SetDrawType(drawType)

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(cfg.SampleRate, cfg.SampleSize)

	spectrum.SetSmoothing(cfg.SmoothFactor)
	spectrum.SetGamma(cfg.Gamma)

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var dispCtx = display.Start(ctx)
	defer display.Stop()

	var endSig = make(chan os.Signal, 2)
	signal.Notify(endSig, os.Interrupt)

	for ch := 0; ch < cfg.ChannelCount; ch++ {
		spectrum.AddStream(audio.SampleBuffers()[ch])
	}

	var barCount = display.Bars(cfg.ChannelCount)
	barCount = spectrum.Recalculate(barCount)

	var tick = time.Now()

	if err := audio.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer audio.Stop()

	var delay = time.NewTimer(0)
	defer delay.Stop()
	<-delay.C

	for {

		delay.Reset(drawDelay - time.Since(tick))

		select {
		case <-ctx.Done():
			return nil
		case <-dispCtx.Done():
			return nil
		case <-endSig:
			return nil
		case <-delay.C:
		}

		tick = time.Now()

		if winWidth := display.Bars(cfg.ChannelCount); barCount != winWidth {
			barCount = winWidth
			barCount = spectrum.Recalculate(barCount)
		}

		if audio.ReadyRead() < cfg.SampleSize {
			continue
		}

		if err := audio.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		spectrum.Process()

		display.Draw(spectrum.Buffers(), barCount)
	}
}
