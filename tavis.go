package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/analysis"
	"github.com/noriah/tavis/fftw"
	"github.com/noriah/tavis/input"
)

// BarType is the type of each bar value
type BarType = float64

// constants for testing
const (
	// DeviceName is the name of the Device we want to listen to
	DeviceName = "VisOut"

	// SampleRate is the rate at which samples are read
	SampleRate = 48000

	LoCutFerq = 220

	HiCutFreq = 6000

	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS = 60

	// NumBars is how many bars we want to show
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

		fftwBuffer fftw.CmplxBuffer
		fftwPlan   *fftw.Plan // fftw plan

		spectrum *analysis.Spectrum

		rootCtx    context.Context
		rootCancel context.CancelFunc

		// last       time.Time // last tick time
		// since      time.Duration
		mainTicker *time.Ticker
	)

	audioInput = input.NewPortaudio(input.Params{
		Device:   "VisOut",
		Channels: ChannelCount,
		Rate:     SampleRate,
		Samples:  SampleSize,
	})

	//FFTW complex data
	fftwBuffer = make(fftw.CmplxBuffer, BufferSize)

	// Our FFTW calculator
	fftwPlan = fftw.New(
		audioInput.Buffer(), fftwBuffer,
		ChannelCount, SampleSize,
		fftw.Estimate)

	// Make a spectrum
	spectrum = analysis.NewSpectrum(SampleRate, SampleSize, ChannelCount)

	// Set it up with our values
	spectrum.Recalculate(NumBars, LoCutFerq, HiCutFreq)

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
			spectrum.Generate(fftwBuffer)
		}

		// fmt.Println(fftwBuffer[0 : NumBars*2])
		fmt.Println(spectrum.Bins())
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
