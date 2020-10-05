package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/display"
	"github.com/noriah/tavis/dsp"
	"github.com/noriah/tavis/input/portaudio"

	"github.com/pkg/errors"
)

type Device struct {
	// Name is the name of the Device we want to listen to
	Name string
	// SampleRate is the rate at which samples are read
	SampleRate float64
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// MonstercatFactor is how much we want to look like Monstercat
	MonstercatFactor float64
	// FalloffWeight is the fall-off weight
	FalloffWeight float64
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
		Name:             "default",
		SampleRate:       44100,
		LoCutFreq:        20,
		HiCutFreq:        22050,
		MonstercatFactor: 2.5,
		FalloffWeight:    0.01,
		BarWidth:         2,
		SpaceWidth:       1,
		TargetFPS:        60,
		ChannelCount:     2,
	}
}

// Run starts to draw the visualizer on the tcell Screen.
func Run(d Device) error {
	var (
		// SampleSize is the number of frames per channel we want per read
		sampleSize = int(d.SampleRate) / d.TargetFPS

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(d.TargetFPS)
	)

	var audioInput = &portaudio.Portaudio{
		DeviceName: d.Name,
		FrameSize:  d.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: d.SampleRate,
	}

	if err := audioInput.Init(); err != nil {
		return err
	}
	defer audioInput.Close()

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(d.SampleRate, sampleSize)

	// TODO(noriah): remove temprorary variables
	var displayChan = make(chan bool, 1)

	var display = display.New()
	defer display.Close()

	var barCount = display.SetWidths(d.BarWidth, d.SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)

	display.Start(displayChan)
	defer display.Stop()

	var endSig = make(chan os.Signal, 3)
	signal.Notify(endSig, os.Interrupt)

	var sets = make([]*dsp.DataSet, d.ChannelCount)

	for xS := range sets {
		sets[xS] = spectrum.DataSet(audioInput.Buffers()[xS])
	}

	audioInput.Start()
	defer audioInput.Stop()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var vIterStart = time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-displayChan:
			return nil
		case <-endSig:
			return nil
		default:
		}

		if vSince := time.Since(vIterStart); vSince < drawDelay {
			time.Sleep(drawDelay - vSince)
		}

		vIterStart = time.Now()

		var winWidth, winHeight = display.Size()
		winHeight /= 2

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)
		}

		if audioInput.ReadyRead() < sampleSize {
			continue
		}

		if err := audioInput.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")

		}

		for xSet := range sets {
			sets[xSet].ExecuteFFTW()

			spectrum.Generate(sets[xSet])

			// dsp.Waves(1.9, ds)
			// dsp.Monstercat(d.MonstercatFactor, sets[xSet])
			dsp.Scale(winHeight, sets[xSet])
			dsp.Falloff(d.FalloffWeight, sets[xSet])
		}

		display.Draw(winHeight, 1, sets...)
	}
}
