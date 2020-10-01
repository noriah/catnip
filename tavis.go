package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/portaudio"
)

// constants for testing
const (

	// DeviceName is the name of the Device we want to listen to
	DeviceName = "VisOut"

	// SampleRate is the rate at which samples are read
	SampleRate = 48000

	//LoCutFerq is the low end of our audio spectrum
	LoCutFerq = 410

	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq = 4000

	// MonstercatFactor is how much do we want to look like monstercat
	MonstercatFactor = 5.75

	// Falloff weight
	FalloffWeight = 0.910

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

	var audioInput = &Portaudio{
		DeviceName: DeviceName,
		FrameSize:  ChannelCount,
		SampleSize: SampleSize,
		SampleRate: SampleRate,
	}

	panicOnError(audioInput.Init())
	defer audioInput.Close()

	var fftwIn = make([]float64, SampleBufferSize)

	audioBuf := audioInput.Buffer()

	// Make a spectrum
	var spectrum = NewSpectrum(SampleRate, SampleSize)

	var sets = make([]*DataSet, ChannelCount)

	for xS := range sets {
		sets[xS] = spectrum.DataSet(fftwIn[xS*SampleSize : (xS+1)*SampleSize])
	}

	var display = NewDisplay()
	defer display.Close()

	var barCount = display.SetWidths(BarWidth, SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)

	var rootCtx, rootCancel = context.WithCancel(context.Background())

	// TODO(noriah): remove temprorary variables
	var displayChan = make(chan bool, 1)

	display.Start(displayChan)
	defer display.Stop()

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

	audioInput.Start()
	defer audioInput.Stop()

	// MAIN LOOP

	var (
		winWidth  int
		winHeight int

		vIterStart time.Time
		vSince     time.Duration

		err       error
		caughtErr interface{}
	)

	var mainTicker = time.NewTicker(DrawDelay)
	defer mainTicker.Stop()

	var catchMe = func() {
		if rec := recover(); rec != nil {
			caughtErr = rec
			rootCancel()
		}
	}

	defer catchMe()

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
					break RunForRest
				}
				err = nil
			}

			deFrame(fftwIn, audioBuf, ChannelCount, SampleSize)

			for _, vSet := range sets {
				vSet.ExecuteFFTW()

				spectrum.Generate(vSet)
				spectrum.Monstercat(MonstercatFactor, vSet)
				spectrum.Scale(winHeight/2, vSet)
				spectrum.Falloff(FalloffWeight, vSet)

			}

			display.Draw(winHeight/2, 1, sets...)
		}
	}

	rootCancel()

	if caughtErr != nil {
		fmt.Println(caughtErr)
	}

	return nil
}

func deFrame(dest []float64, src []float32, count, size int) {

	// This "fix" is because the portaudio interface we are using does not
	// work properly. I have to de-interleave the array
	for xBuf, xOffset := 0, 0; xOffset < count*size; xOffset += size {
		for xCnt := 0; xCnt < size; xCnt++ {
			dest[xBuf] = float64(src[xOffset+xCnt])
			xBuf++
		}
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
