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

type channel struct {
	bins *dsp.BinSet
	n2s3 *dsp.N2S3State
}

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

	var barCount = display.Bars(cfg.ChannelCount)

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(cfg.SampleRate, cfg.SampleSize)

	// Set it up with our values
	barCount = spectrum.Recalculate(barCount)

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var dispCtx = display.Start(ctx)
	defer display.Stop()

	var endSig = make(chan os.Signal, 2)
	signal.Notify(endSig, os.Interrupt)

	var channels = make([]channel, cfg.ChannelCount)

	var chanBins = make([][]float64, cfg.ChannelCount)

	for ch, bufs := 0, audio.SampleBuffers(); ch < len(chanBins); ch++ {
		channels[ch] = channel{
			bins: spectrum.BinSet(bufs[ch]),
			n2s3: dsp.NewN2S3State(cfg.SampleSize),
		}

		chanBins[ch] = channels[ch].bins.Bins()
	}

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

		var winWidth = display.Bars(cfg.ChannelCount)

		if barCount != winWidth {
			barCount = winWidth
			barCount = spectrum.Recalculate(barCount)
		}

		if audio.ReadyRead() < cfg.SampleSize {
			continue
		}

		if err := audio.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		for ch := range channels {
			spectrum.Generate(channels[ch].bins)

			// nora's not so special smoother (n2s3)
			dsp.N2S3(chanBins[ch], barCount, channels[ch].n2s3, cfg.SmoothFactor, cfg.SmoothResponse)

			// dsp.Monstercat(chanBins[ch], barCount, 1.95)

		}

		display.Draw(chanBins, barCount)
	}
}
