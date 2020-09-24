package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/runningwild/go-fftw/fftw"
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
	SampleSize = int(SampleRate) / TargetFPS
	BufferSize = SampleSize * ChannelCount
	DrawDelay  = time.Second / TargetFPS
)

type SampleType = float32

// Run does the run things
func Run() error {
	var err error

	// MAIN LOOP PREP

	var (
		endSig chan os.Signal

		readKickChan  chan bool
		readReadyChan chan bool

		idx    int       // general use index
		sample rawSample // general use sample

		fftBuffer *fftw.Array2 // fftw input array
		fftPlan   *fftw.Plan   // fftw plan

		rootCtx    context.Context
		rootCancel context.CancelFunc

		last       time.Time // last tick time
		since      time.Duration
		mainTicker *time.Ticker
	)

	endSig = make(chan os.Signal, 3)
	signal.Notify(endSig, os.Interrupt)

	readKickChan = make(chan bool, 1)
	readReadyChan = make(chan bool, 1)

	fftBuffer = &fftw.Array2{
		N:     [...]int{ChannelCount, SampleSize},
		Elems: make([]complex128, BufferSize),
	}

	fftPlan = fftw.NewPlan2(
		fftBuffer, fftBuffer,
		fftw.Forward, fftw.Estimate)

	rootCtx, rootCancel = context.WithCancel(context.Background())

	// Handle fanout of cancel
	go func() {
		select {
		case <-rootCtx.Done():
		case <-endSig:
		}

		rootCancel()
	}()

	// MAIN LOOP

	last = time.Now()

	mainTicker = time.NewTicker(DrawDelay)

	// kickstart the read process
	readKickChan <- struct{}{}

RunForRest: // , run!!!
	for {
		since = time.Since(last)

		if since > DrawDelay {
			fmt.Print("slow loop!")
		}

		fmt.Println(since)

		select {
		case <-rootCtx.Done():
			break RunForRest
		case last = <-mainTicker.C:
		}

		select {
		case <-rootCtx.Done():
			break RunForRest
		case <-readReadyChan:
		}

		for idx, sample = range rawBuffer {
			fftBuffer.Elems[idx] = complex(float64(sample), 0)
		}

		select {
		case <-rootCtx.Done():
			break RunForRest
		case readKickChan <- struct{}{}:
		}

		fftPlan.Execute()

	}

	rootCancel()

	// CLEANUP

	mainTicker.Stop()

	fftPlan.Destroy()

	return nil
}
