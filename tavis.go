package tavis

import (
	"context"
	"fmt"
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

	// MAIN LOOP PREP

	var (
		err error

		fftwBuffer []complex128
		fftwPlan   *fftw.Plan

		winWidth  int
		winHeight int

		pulseError error

		vIterStart time.Time
		vSince     time.Duration

		mainTicker *time.Ticker
	)

	var audioInput = &Portaudio{
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
	var spectrum = NewSpectrum(SampleRate, ChannelCount, SampleSize)

	for xSet, vSet := range spectrum.DataSets() {
		vSet.DataBuf = fftwBuffer[xSet*FFTWDataSize : (xSet+1)*FFTWDataSize]
	}

	var display = NewDisplay(spectrum.DataSets())

	var barCount = display.SetWidths(BarWidth, SpaceWidth)

	// Set it up with our values
	spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)

	var rootCtx, rootCancel = context.WithCancel(context.Background())

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

			deFrame(tmpBuf, audioBuf, ChannelCount, SampleSize)

			fftwPlan.Execute()

			spectrum.Generate()

			spectrum.Monstercat(MonstercatFactor)

			// winHeight = winHeight / 2
			spectrum.Scale(winHeight / 2)

			spectrum.Falloff(FalloffWeight)

			display.Draw(winHeight/2, 1)
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
		fmt.Println(pulseError)
	}

	return nil
}

func deFrame(dest []float64, src []float32, count, size int) {

	// This "fix" is because the portaudio interface we are using does not
	// work properly. I have to de-interleave the array
	for xBuf, xOffset := 0, 0; xOffset < count*size; xOffset += size {
		for xCnt := 0; xCnt < size; xCnt++ {
			dest[xOffset+xCnt] = float64(src[xBuf])
			xBuf++
		}
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
