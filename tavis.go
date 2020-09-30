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
	SampleRate = 48000

	LoCutFerq = 410

	HiCutFreq = 4000

	MonstercatFactor = 3.64

	FalloffWeight = 0.910

	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS = 100

	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount = 2
)

// calculated constants
const (
	// SampleSize is the number of frames per channel we want per read
	SampleSize = SampleRate / TargetFPS

	// BufferSize is the total size of our buffer (SampleSize * FrameSize)
	BufferSize = SampleSize * ChannelCount

	// DrawDelay is the time we wait between ticks to draw.
	DrawDelay = time.Second / TargetFPS
)

// Run does the run things
func Run() error {

	// MAIN LOOP PREP

	var (
		err error

		audioInput *Portaudio

		fftwBuffer []fftw.ComplexType
		fftwPlan   *fftw.Plan // fftw plan

		spectrum *Spectrum

		display *Display

		rootCtx    context.Context
		rootCancel context.CancelFunc

		barCount int

		winWidth  int
		winHeight int

		// last       time.Time // last tick time
		// since      time.Duration
		mainTicker *time.Ticker
	)

	audioInput = &Portaudio{
		DeviceName: DeviceName,
		FrameSize:  ChannelCount,
		SampleSize: SampleSize,
		SampleRate: SampleRate,
	}

	panicOnError(audioInput.Init())

	tmpBuf := make([]float64, BufferSize)

	//FFTW complex data
	fftwBuffer = make([]fftw.ComplexType, (SampleSize/2+1)*ChannelCount)

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
		sampleDataSize: SampleSize/2 + 1,
		frameSize:      ChannelCount,
		DataBuf:        fftwBuffer,
	}

	panicOnError(spectrum.Init())

	display = &Display{
		DataSets: spectrum.DataSets(),
	}

	panicOnError(display.Init())

	barCount = display.SetWidths(2, 1)

	// Set it up with our values
	spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)

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

		select {
		case <-rootCtx.Done():
			break RunForRest
		default:
		}

		winWidth, winHeight = display.Size()

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)
		}

		if audioInput.ReadyRead() >= SampleSize {
			if err = audioInput.Read(rootCtx); err != nil {
				if err != portaudio.InputOverflowed {
					panic(err)
				}
			}

			// This "fix" is because the portaudio interface Im using does not
			// work properly. I have rebuild the array for them
			for s := 0; s < ChannelCount; s++ {
				for x := 0; x < SampleSize; x++ {
					// tmpBuf[x+(SampleSize*s)] = float64(audioBuf[(x*ChannelCount)+s])
					tmpBuf[(x*ChannelCount)+s] = float64(audioBuf[(x*ChannelCount)+s])
				}
			}
			fftwPlan.Execute()

			winHeight = (winHeight / 2)

			spectrum.Generate()

			spectrum.Monstercat(MonstercatFactor)

			spectrum.Scale(winHeight)

			spectrum.Falloff(FalloffWeight)
			// go display.Draw()
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

	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
