package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	fftw "github.com/noriah/tavis/fftw"

	"github.com/gdamore/tcell/v2"
)

// constants for testing
const (
	DeviceName   = "VisOut"
	SampleRate   = 48000
	TargetFPS    = 60
	ChannelCount = 2
)

// calculated constants
const (
	SampleSize = 256
	BufferSize = SampleSize * ChannelCount
	DrawDelay  = time.Second / TargetFPS
)

// Run does the run things
func Run() error {
	var err error

	// MAIN LOOP PREP

	var (
		audioInput *Portaudio

		screen tcell.Screen

		rawBuffer SampleBuffer

		fftwBuffer []fftw.FftwComplexType
		fftwPlan   *fftw.Plan // fftw plan

		rootCtx    context.Context
		rootCancel context.CancelFunc

		// last       time.Time // last tick time
		// since      time.Duration
		mainTicker *time.Ticker
	)

	audioInput = &Portaudio{
		DeviceName: "VisOut",
		FrameSize:  ChannelCount,
		SampleRate: SampleRate,
		SampleSize: SampleSize,
	}

	panicOnError(audioInput.Init())

	rawBuffer = make(SampleBuffer, BufferSize)
	fftwBuffer = make([]fftw.FftwComplexType, BufferSize)

	fftwPlan = fftw.New(
		rawBuffer, fftwBuffer, ChannelCount, SampleSize,
		fftw.Forward, fftw.Estimate)

	if screen, err = tcell.NewScreen(); err != nil {
		panic(err)
	}

	panicOnError(screen.Init())

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

		if audioInput.Read(rootCtx, rawBuffer) == 0 {
			// fmt.Println("what happened!")
		}

		fftwPlan.Execute()

		fmt.Println(rawBuffer)

		// since = time.Since(last)
		// if since > DrawDelay {
		// 	fmt.Print("slow loop!\n")
		// }
	}

	rootCancel()

	// CLEANUP

	audioInput.Stop()

	mainTicker.Stop()

	screen.Fini()

	audioInput.Close()

	fftwPlan.Destroy()

	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
