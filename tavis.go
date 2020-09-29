package tavis

import (
	"context"
	"os"
	"os/signal"
	"time"
	"unsafe"

	"github.com/noriah/tavis/fftw"
	"github.com/noriah/tavis/portaudio"
)

// BarType is the type of each bar value
type BarType = float64

// BarBuffer is a slice of CmplxType
type BarBuffer []BarType

// Ptr returns a pointer for use with CGO
func (cb BarBuffer) Ptr(n ...int) unsafe.Pointer {
	if len(n) > 0 {
		return unsafe.Pointer(&cb[n[0]])
	}

	return unsafe.Pointer(&cb[0])
}

// constants for testing
const (
	// DeviceName is the name of the Device we want to listen to
	DeviceName = "VisOut"

	// SampleRate is the rate at which samples are read
	SampleRate = 48000

	LoCutFerq = 220

	HiCutFreq = 6000

	MonstercatFactor = 3.64

	FalloffWeight = 0.910

	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS = 60

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

		fftwBuffer []complex128
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

	audioInput = NewPortaudio(Params{
		Device:   DeviceName,
		Channels: ChannelCount,
		Rate:     SampleRate,
		Samples:  SampleSize,
	})

	tmpBuf := make(BarBuffer, BufferSize)

	//FFTW complex data
	fftwBuffer = make([]complex128, BufferSize)

	audioBuf := audioInput.Buffer()

	// Our FFTW calculator
	fftwPlan = fftw.New(
		tmpBuf, fftwBuffer,
		ChannelCount, SampleSize,
		fftw.Forward, fftw.Estimate)

	// Make a spectrum
	spectrum = &Spectrum{
		sampleRate: SampleRate,
		sampleSize: SampleSize,
		frameSize:  ChannelCount,
		dataBuf:    fftwBuffer,
	}

	panicOnError(spectrum.Init())

	display = &Display{
		dataBuf: spectrum.workBins,
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
		// last = time.Now()
		select {
		case <-rootCtx.Done():
			break RunForRest
		default:
		}

		if audioInput.ReadyRead() >= SampleSize {
			if err = audioInput.Read(rootCtx); err != nil {
				if err != portaudio.InputOverflowed {
					panic(err)
				}
			}

			for x := 0; x < len(tmpBuf); x++ {
				tmpBuf[x] = float64(audioBuf[x])
			}
		}
		fftwPlan.Execute()

		winWidth, winHeight = display.Size()

		if barCount != winWidth {
			barCount = winWidth
			spectrum.Recalculate(barCount, LoCutFerq, HiCutFreq)
		}
		winHeight = (winHeight / 2)

		spectrum.Generate()
		spectrum.Monstercat(MonstercatFactor)
		spectrum.Scale(winHeight)
		spectrum.Falloff(FalloffWeight)

		display.Draw()
		// fmt.Println(spectrum.Bins())

		// since = time.Since(last)
		// if since > DrawDelay {
		// 	fmt.Print("slow loop!\n", since)
		// }
	}

	rootCancel()

	// CLEANUP

	audioInput.Stop()

	display.Stop()

	display.Close()

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
