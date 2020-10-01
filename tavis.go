package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/display"
	"github.com/noriah/tavis/dsp"
	"github.com/noriah/tavis/dsp/n2s3"
	"github.com/noriah/tavis/input"
	"github.com/pkg/errors"
)

type Device struct {
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
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
}

// NewZeroDevice creates a new Device with the default variables.
func NewZeroDevice() Device {
	return Device{
		SampleRate:   44100,
		LoCutFreq:    20,
		HiCutFreq:    22050,
		BarWidth:     2,
		SpaceWidth:   1,
		TargetFPS:    60,
		ChannelCount: 2,
	}
}

// Run starts to draw the visualizer on the tcell Screen.
func (d Device) Run() error {
	var (
		// SampleSize is the number of frames per channel we want per read
		sampleSize = int(d.SampleRate / float64(d.TargetFPS))

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(d.TargetFPS)
	)

	s, err := d.InputBackend.Start(input.SessionConfig{
		Device:     d.InputDevice,
		FrameSize:  d.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: d.SampleRate,
	})
	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	var display = display.New()
	defer display.Close()

	var barCount = display.SetWidths(d.BarWidth, d.SpaceWidth)

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(d.SampleRate, sampleSize)

	// Set it up with our values
	spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)

	// TODO(noriah): remove temprorary variables
	var displayChan = make(chan bool, 2)

	display.Start(displayChan)
	defer display.Stop()

	var endSig = make(chan os.Signal, 3)
	signal.Notify(endSig, os.Interrupt)

	var bufs = s.SampleBuffers()
	var sets = make([]*dsp.DataSet, d.ChannelCount)
	for xS := range sets {
		sets[xS] = spectrum.DataSet(bufs[xS])
	}

	if err := s.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer s.Stop()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var ticker = time.NewTicker(drawDelay)
	defer ticker.Stop()

	var tick time.Time

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-displayChan:
			return nil
		case <-endSig:
			return nil
		case tick = <-ticker.C:
		}

		var winWidth, winHeight = display.Size()
		winHeight /= 2

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)
		}

		if s.ReadyRead() < sampleSize {
			continue
		}

		if err := s.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		for _, vSet := range sets {
			spectrum.Generate(vSet)

			n2s3.N2S3(vSet.Bins(), vSet.Len(), tick, vSet.N2S3State)

			dsp.Scale(vSet.Bins(), vSet.Len(), winHeight, vSet.ScaleState)
		}

		display.Draw(winHeight, 1, sets...)
	}
}
