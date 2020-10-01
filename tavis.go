package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/fftw"
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
		LoCutFreq:        20,
		HiCutFreq:        8000,
		MonstercatFactor: 8.75,
		FalloffWeight:    0.895,
		BarWidth:         2,
		SpaceWidth:       1,
		TargetFPS:        60,
		ChannelCount:     2,
	}
}

// calculated constants
const ()

// Run starts to draw the visualizer on the tcell Screen.
func (d Device) Run() error {
	var (
		// SampleSize is the number of frames per channel we want per read
		sampleSize = int(d.SampleRate / float64(d.TargetFPS))

		// FFTWDataSize is the number of data points in an fftw data set return
		fftwDataSize = (sampleSize / 2) + 1

		// BufferSize is the total size of our buffer (SampleSize * FrameSize)
		sampleBufferSize = sampleSize * d.ChannelCount

		// FFTWBufferSize is the total size of our fftw complex128 buffer
		fftwBufferSize = fftwDataSize * d.ChannelCount

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(d.TargetFPS)
	)

	// MAIN LOOP PREP

	var (
		barCount int

		xSet int
		xBuf int

		winWidth  int
		winHeight int

		vIterStart time.Time
		vSince     time.Duration
	)

	audioInput := &Portaudio{
		DeviceName: d.Name,
		FrameSize:  d.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: d.SampleRate,
	}

	if err := audioInput.Init(); err != nil {
		return err
	}

	defer audioInput.Close()

	tmpBuf := make([]float64, sampleBufferSize)

	//FFTW complex data
	fftwBuffer := make([]complex128, fftwBufferSize)

	audioBuf := audioInput.Buffer()

	// Our FFTW calculator
	fftwPlan := fftw.New(
		tmpBuf, fftwBuffer,
		d.ChannelCount, sampleSize,
		fftw.Estimate,
	)

	defer fftwPlan.Destroy()

	// Make a spectrum
	spectrum := &Spectrum{
		sampleRate:     d.SampleRate,
		sampleSize:     sampleSize,
		sampleDataSize: fftwDataSize,
		frameSize:      d.ChannelCount,
		DataBuf:        fftwBuffer,
	}

	if err := spectrum.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize spectrum")
	}

	display := &Display{
		DataSets: spectrum.DataSets(),
	}

	if err := display.Init(); err != nil {
		return errors.Wrap(err, "failed to create display")
	}

	defer display.Close()
	display.SetWidths(d.BarWidth, d.SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(1, d.LoCutFreq, d.HiCutFreq)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO(noriah): remove temprorary variables
	displayChan := make(chan bool, 1)

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

	// MAIN LOOP

	display.Start(displayChan)
	defer display.Stop()

	audioInput.Start()
	defer audioInput.Stop()

	mainTicker := time.NewTicker(drawDelay)
	defer mainTicker.Stop()

RunForRest: // , run!!!
	for range mainTicker.C {
		if vSince = time.Since(vIterStart); vSince < drawDelay {
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

		if audioInput.ReadyRead() >= sampleSize {
			if err := audioInput.Read(ctx); err != nil {
				if err != portaudio.InputOverflowed {
					return errors.Wrap(err, "failed to read audio input")
				}
			}

			// This "fix" is because the portaudio interface we are using does not
			// work properly. I have to de-interleave the array
			for xSet = 0; xSet < d.ChannelCount; xSet++ {
				for xBuf = 0; xBuf < sampleSize; xBuf++ {
					tmpBuf[xBuf+(sampleSize*xSet)] = float64(audioBuf[(xBuf*d.ChannelCount)+xSet])
				}
			}

			fftwPlan.Execute()

			spectrum.Generate()

			spectrum.Monstercat(d.MonstercatFactor)

			// winHeight = winHeight / 2
			spectrum.Scale(winHeight / 2)

			spectrum.Falloff(d.FalloffWeight)

			display.Draw(winHeight / 2)
		}
	}

	return nil
}
