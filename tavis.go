package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/fftw"
	"github.com/noriah/tavis/portaudio"
)

// constants for testing
const (
	// DeviceName is the name of the Device we want to listen to
	DeviceName = "VisOut"

	// SampleRate is the rate at which samples are read
	SampleRate = 96000

	//LoCutFerq is the low end of our audio spectrum
	LoCutFerq = 20

	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq = 8000

	// MonstercatFactor is how much do we want to look like monstercat
	MonstercatFactor = 8.75

	// Falloff weight
	FalloffWeight = 0.895

	// BarWidth is the width of bars, in columns
	BarWidth = 2

	// SpaceWidth is the width of spaces, in columns
	SpaceWidth = 1

	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS = 100

	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount = 2
)

// calculated constants
const (
	// SampleSize is the number of frames per channel we want per read
	SampleSize = SampleRate / TargetFPS

	// FFTWDataSize is the number of data points in an fftw data set return
	FFTWDataSize = (SampleSize / 2) + 1

	// BufferSize is the total size of our buffer (SampleSize * FrameSize)
	SampleBufferSize = SampleSize * ChannelCount

	// FFTWBufferSize is the total size of our fftw complex128 buffer
	FFTWBufferSize = FFTWDataSize * ChannelCount

	// DrawDelay is the time we wait between ticks to draw.
	DrawDelay = time.Second / TargetFPS
)

// Run does the run things
func Run() error {

	// MAIN LOOP PREP

	var (
		err error

		audioInput *Portaudio

		fftwBuffer []complex128
		fftwPlan   *fftw.Plan

		spectrum *Spectrum

		display *Display

		rootCtx    context.Context
		rootCancel context.CancelFunc

		barCount int

		xSet int
		xBuf int

		winWidth  int
		winHeight int

		pulseError error

		vIterStart time.Time
		vSince     time.Duration

		mainTicker *time.Ticker
	)

	audioInput = &Portaudio{
		DeviceName: DeviceName,
		FrameSize:  ChannelCount,
		SampleSize: SampleSize,
		SampleRate: SampleRate,
	}

	panicOnError(audioInput.Init())

	tmpBuf := make([]float64, SampleBufferSize)

	//FFTW complex data
	fftwBuffer = make([]complex128, FFTWBufferSize)

	audioBuf := audioInput.Buffer()

	// Our FFTW calculator
	fftwPlan = fftw.New(
		tmpBuf, fftwBuffer,
		ChannelCount, SampleSize,
		fftw.Estimate)

	// Make a spectrum
	spectrum = &Spectrum{
		sampleRate:     SampleRate,
		sampleSize:     SampleSize,
		sampleDataSize: FFTWDataSize,
		frameSize:      ChannelCount,
		DataBuf:        fftwBuffer,
	}

	panicOnError(spectrum.Init())

	display = &Display{
		DataSets: spectrum.DataSets(),
	}

	panicOnError(display.Init())

	display.SetWidths(BarWidth, SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(1, LoCutFerq, HiCutFreq)

	rootCtx, rootCancel = context.WithCancel(context.Background())

	// TODO(noriah): remove temprorary variables
	displayChan := make(chan bool, 1)

	// Handle fanout of cancel
	go func() {

		var endSig chan os.Signal

		endSig = make(chan os.Signal, 3)
		signal.Notify(endSig, os.Interrupt)

		select {
		case <-rootCtx.Done():
		case <-displayChan:
		case <-endSig:
		}

		rootCancel()
	}()

	// MAIN LOOP

	display.Start(displayChan)

	audioInput.Start()

	mainTicker = time.NewTicker(DrawDelay)

RunForRest: // , run!!!
	for range mainTicker.C {
		if vSince = time.Since(vIterStart); vSince < DrawDelay {
			time.Sleep(DrawDelay - vSince)
		}

		select {
		case <-rootCtx.Done():
			break RunForRest
		default:
		}

		vIterStart = time.Now()

		winWidth, winHeight = display.Size()

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)
		}

		if audioInput.ReadyRead() >= SampleSize {
			if err = audioInput.Read(rootCtx); err != nil {
				if err != portaudio.InputOverflowed {
					pulseError = err
					break RunForRest
				}
			}

			// This "fix" is because the portaudio interface we are using does not
			// work properly. I have to de-interleave the array
			for xSet = 0; xSet < ChannelCount; xSet++ {
				for xBuf = 0; xBuf < SampleSize; xBuf++ {
					tmpBuf[xBuf+(SampleSize*xSet)] = float64(audioBuf[(xBuf*ChannelCount)+xSet])
				}
			}

			fftwPlan.Execute()

			spectrum.Generate()

			spectrum.Monstercat(MonstercatFactor)

			// winHeight = winHeight / 2
			spectrum.Scale(winHeight / 2)

			spectrum.Falloff(FalloffWeight)

			display.Draw()
		}
	}

	rootCancel()

	// CLEANUP

	audioInput.Stop()

	audioInput.Close()

	display.Stop()

	display.Close()

	mainTicker.Stop()

	fftwPlan.Destroy()

	if pulseError != nil {
	}

	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
