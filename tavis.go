package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/fftw"
	"github.com/noriah/tavis/input"
)

// constants for testing
const (
	// DeviceName is the name of the Device we want to listen to
	DeviceName = "VisOut"

	// SampleRate is the rate at which samples are read
	SampleRate = 48000

	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS = 60

	// MaxBars is the maximum number of bars we will display
	MaxBars = 512

	// NumBars is how many bars we start with
	NumBars = 128

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

		audioInput *input.Portaudio

		rawBuffer input.SampleBuffer

		fftwBuffer fftw.CmplxBuffer
		fftwPlan   *fftw.Plan // fftw plan

		barBuffer BarBuffer

		spectrum *Spectrum

		rootCtx    context.Context
		rootCancel context.CancelFunc

		// last       time.Time // last tick time
		// since      time.Duration
		mainTicker *time.Ticker
	)

	rawBuffer = make(input.SampleBuffer, BufferSize)

	audioInput = &input.Portaudio{
		DeviceName:   "VisOut",
		FrameSize:    ChannelCount,
		SampleRate:   SampleRate,
		SampleSize:   SampleSize,
		SampleBuffer: rawBuffer,
	}

	panicOnError(audioInput.Init())

	fftwBuffer = make(fftw.CmplxBuffer, BufferSize)

	fftwPlan = fftw.New(
		rawBuffer, fftwBuffer, ChannelCount, SampleSize,
		fftw.Estimate)

	barBuffer = make(BarBuffer, MaxBars)

	spectrum = &Spectrum{
		FrameSize:  ChannelCount,
		SampleRate: SampleRate,
		SampleSize: SampleSize,
		BarBuffer:  barBuffer,
		Data:       fftwBuffer,
	}

	panicOnError(spectrum.Init())

	spectrum.Recalculate(NumBars, 400, 6000)

	rootCtx, rootCancel = context.WithCancel(context.Background())

	// Handle fanout of cancel
	go func() {

		var endSig chan os.Signal

		endSig = make(chan os.Signal, 3)
		signal.Notify(endSig, os.Interrupt)

		select {
		case <-rootCtx.Done():
		case <-endSig:
		}

		rootCancel()
	}()

	// MAIN LOOP

	audioInput.Start()

	mainTicker = time.NewTicker(DrawDelay)

RunForRest: // , run!!!
	for range mainTicker.C {
		// last = time.Now()
		select {
		case <-rootCtx.Done():
			break RunForRest
		default:
		}

		if audioInput.ReadyRead() >= SampleSize {
			if err = audioInput.Read(rootCtx); err != nil {
				fmt.Println("what happened!", err)
			}

			fftwPlan.Execute()
			spectrum.Generate(30, 1.6)
		}

		// fmt.Println(fftwBuffer[0 : NumBars*2])
		fmt.Println(barBuffer[0 : NumBars*2])
		// spectrum.Print()

		// since = time.Since(last)
		// if since > DrawDelay {
		// 	fmt.Print("slow loop!\n", since)
		// }
	}

	rootCancel()

	// CLEANUP

	audioInput.Stop()

	mainTicker.Stop()

	audioInput.Close()

	fftwPlan.Destroy()

	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
