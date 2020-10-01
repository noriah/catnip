package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/portaudio"
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
		LoCutFreq:        410,
		HiCutFreq:        8000,
		MonstercatFactor: 3.5,
		FalloffWeight:    0.912,
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

		// BufferSize is the total size of our buffer (SampleSize * FrameSize)
		sampleBufferSize = sampleSize * d.ChannelCount

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(d.TargetFPS)

		winWidth  int
		winHeight int

		vIterStart time.Time
	)

	var audioInput = &Portaudio{
		DeviceName: d.Name,
		FrameSize:  d.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: d.SampleRate,
	}

	if err := audioInput.Init(); err != nil {
		return err
	}

	defer audioInput.Close()

	var fftwIn = make([][]float64, d.ChannelCount)

	var audioBuf = audioInput.Buffer()

	// Make a spectrum
	var spectrum = NewSpectrum(d.SampleRate, sampleSize)

	var sets = make([]*DataSet, d.ChannelCount)

	for xS := range sets {
		fftwIn[xS] = make([]float64, sampleSize)
		sets[xS] = spectrum.DataSet(fftwIn[xS])
	}

	var display = NewDisplay()
	defer display.Close()

	var barCount = display.SetWidths(d.BarWidth, d.SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// TODO(noriah): remove temprorary variables
	var displayChan = make(chan bool, 1)

	display.Start(displayChan)
	defer display.Stop()

	// Handle fanout of cancel
	go func() {
		endSig := make(chan os.Signal, 3)
		signal.Notify(endSig, os.Interrupt)

		select {
		case <-ctx.Done():
		case <-displayChan:
		case <-endSig:
		}

		cancel()
	}()

	display.Start(displayChan)
	defer display.Stop()

	audioInput.Start()
	defer audioInput.Stop()

	mainTicker := time.NewTicker(drawDelay)
	defer mainTicker.Stop()

RunForRest: // , run!!!
	for range mainTicker.C {
		if vSince := time.Since(vIterStart); vSince < drawDelay {
			time.Sleep(drawDelay - vSince)
		}

		select {
		case <-ctx.Done():
			break RunForRest
		default:
		}

		vIterStart = time.Now()

		winWidth, winHeight = display.Size()

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)
		}

		if audioInput.ReadyRead() >= sampleBufferSize {
			if err := audioInput.Read(ctx); err != nil {
				if err != portaudio.InputOverflowed {
					return errors.Wrap(err, "failed to read audio input")
				}
				err = nil
			}

			deFrame(fftwIn, audioBuf)

			for _, vSet := range sets {
				vSet.ExecuteFFTW()

				spectrum.Generate(vSet)
				spectrum.Monstercat(d.MonstercatFactor, vSet)
				spectrum.Scale(winHeight/2, vSet)
				spectrum.Falloff(d.FalloffWeight, vSet)

			}

			display.Draw(winHeight/2, 1, sets...)
		}
	}

	return nil
}

func deFrame(dest [][]float64, src []float32) {

	// This "fix" is because the portaudio interface we are using does not
	// work properly. I have to de-interleave the array
	for xSet, sets := 0, len(dest); xSet < sets; xSet++ {
		for xSmpl := range dest[xSet] {
			dest[xSet][xSmpl] = float64(src[(xSmpl*sets)+xSet])
		}
	}
}
