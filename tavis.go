package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/display"
	"github.com/noriah/tavis/dsp"
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

	// SampleSize is the number of frames per channel we want per read
	var sampleSize = int(cfg.SampleRate / float64(cfg.TargetFPS))

	// DrawDelay is the time we wait between ticks to draw.
	var drawDelay = time.Second / time.Duration(cfg.TargetFPS)

	var audio, err = cfg.InputBackend.Start(input.SessionConfig{
		Device:     cfg.InputDevice,
		FrameSize:  cfg.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: cfg.SampleRate,
	})

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	var display = display.New(cfg.SampleRate, sampleSize)
	defer display.Close()

	var barCount = display.SetWidths(cfg.BarWidth, cfg.SpaceWidth)

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(cfg.SampleRate, sampleSize, cfg.MaxBins)

	// Set it up with our values
	barCount = spectrum.Recalculate(barCount, cfg.LoCutFreq, cfg.HiCutFreq)

	var channels = make([]channel, cfg.ChannelCount)

	var chanBins = make([][]float64, cfg.ChannelCount)

	for ch, bufs := 0, audio.SampleBuffers(); ch < len(chanBins); ch++ {
		channels[ch] = channel{
			bins: spectrum.BinSet(bufs[ch]),
			n2s3: dsp.NewN2S3State(cfg.MaxBins),
		}

		chanBins[ch] = channels[ch].bins.Bins()
	}
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var dispCtx = display.Start(ctx)
	defer display.Stop()

	var endSig = make(chan os.Signal, 1)
	signal.Notify(endSig, os.Interrupt)

	if err := audio.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer audio.Stop()

	var tick = time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dispCtx.Done():
			return nil
		case <-endSig:
			return nil
		default:
		}

		if since := time.Since(tick); since < drawDelay {
			time.Sleep(drawDelay - since)
		}

		tick = time.Now()

		var winWidth = display.Bars()

		if barCount != winWidth {
			barCount = winWidth
			barCount = spectrum.Recalculate(barCount, cfg.LoCutFreq, cfg.HiCutFreq)
		}

		if audio.ReadyRead() < sampleSize {
			continue
		}

		if err := audio.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		for ch := range channels {
			spectrum.Generate(channels[ch].bins)

			// nora's not so special smoother (n2s3)
			dsp.N2S3(chanBins[ch], barCount,
				channels[ch].n2s3, cfg.SmoothFactor, cfg.SmoothResponse)
		}

		display.Draw(chanBins, barCount, cfg.BaseThick)
	}
}

func sanitizeConfig(cfg *Config) error {

	switch {
	case cfg.SmoothFactor > 100.0:
		cfg.SmoothFactor = 1.0
	case cfg.SmoothFactor <= 0.0:
		cfg.SmoothFactor = 0.00001
	default:
		cfg.SmoothFactor /= 100.0
	}

	switch {
	case cfg.SmoothResponse < 0.1:
		cfg.SmoothResponse = 0.1
	default:
	}

	return nil
}
