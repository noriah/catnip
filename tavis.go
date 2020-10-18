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

// Config is a temporary struct to define parameters
type Config struct {
	// InputBackend is the backend that the input belongs to
	InputBackend input.Backend
	// InputDevice is the device we want to listen to
	InputDevice input.Device
	// SampleRate is the rate at which samples are read
	SampleRate float64
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// SmoothFactor factor of smooth
	SmoothFactor float64
	// SmoothResponse response value
	SmoothResponse float64
	// BaseThick number of cells wide/high the base is
	BaseThick int
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
	// MaxBins maximum number of bins
	MaxBins int
}

// NewZeroConfig returns a zero config
func NewZeroConfig() Config {
	return Config{
		SampleRate:     44100,
		LoCutFreq:      20,
		HiCutFreq:      22050,
		SmoothFactor:   52.5,
		SmoothResponse: 43.5,
		BaseThick:      1,
		BarWidth:       2,
		SpaceWidth:     1,
		TargetFPS:      60,
		ChannelCount:   2,
		MaxBins:        256,
	}
}

type channel struct {
	bins *dsp.BinSet
	n2s3 *dsp.N2S3State
}

// Run starts to draw the visualizer on the tcell Screen.
func tavis(cfg Config) error {
	var (
		// SampleSize is the number of frames per channel we want per read
		sampleSize = int(cfg.SampleRate / float64(cfg.TargetFPS))

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(cfg.TargetFPS)
	)

	var source, err = cfg.InputBackend.Start(input.SessionConfig{
		Device:     cfg.InputDevice,
		FrameSize:  cfg.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: cfg.SampleRate,
	})

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

	if err := source.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer source.Stop()

	var bufs = source.SampleBuffers()

	var channels = make([]*channel, cfg.ChannelCount)

	var chanBins = make([][]float64, cfg.ChannelCount)

	for ch := 0; ch < cfg.ChannelCount; ch++ {
		channels[ch] = &channel{
			bins: spectrum.BinSet(bufs[ch]),
			n2s3: dsp.NewN2S3State(cfg.MaxBins),
		}

		chanBins[ch] = channels[ch].bins.Bins()
	}
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var dispCtx = display.Start(ctx)
	defer display.Stop()

	var endSig = make(chan os.Signal, 3)
	signal.Notify(endSig, os.Interrupt)

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

		if source.ReadyRead() < sampleSize {
			continue
		}

		if err := source.Read(ctx); err != nil {
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
