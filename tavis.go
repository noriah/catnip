package tavis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/runningwild/go-fftw/fftw"
)

// constants for testing
const (
	DeviceName   = "VisOut"
	SampleRate   = 44100
	TargetFPS    = 60
	ChannelCount = 2
)

// calculated constants
const (
	SampleSize = int(SampleRate) / TargetFPS
	BufferSize = SampleSize * ChannelCount
	DrawDelay  = time.Second / TargetFPS
)

// errors
var (
	ErrBadDevice error = errors.New("device not found")
)

// alias our preferred channel for testing purposes
type rawSample = float32

// Run does the run things
func Run() error {
	var err error

	// PORTAUDIO THINGS

	if err = portaudio.Initialize(); err != nil {
		return err
	}

	var devices []*portaudio.DeviceInfo

	if devices, err = portaudio.Devices(); err != nil {
		return err
	}

	var device *portaudio.DeviceInfo

	for idx := 0; idx < len(devices); idx++ {
		if strings.Compare(devices[idx].Name, DeviceName) == 0 {
			device = devices[idx]
			break
		}
	}

	if device == nil {
		return ErrBadDevice
	}

	var (
		rawBuffer []rawSample       // raw sample buffer
		paStream  *portaudio.Stream // portaudio stream
	)

	rawBuffer = make([]rawSample, BufferSize)

	if paStream, err = portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device:   device,
				Channels: ChannelCount,
				Latency:  device.DefaultLowInputLatency,
			},
			SampleRate:      SampleRate,
			FramesPerBuffer: SampleSize,
			Flags:           portaudio.ClipOff,
		}, &rawBuffer); err != nil {
		return err
	}

	// MAIN LOOP PREP

	var (
		endSig chan os.Signal

		readKickChan  chan struct{}
		readReadyChan chan struct{}

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

	readKickChan = make(chan struct{})
	readReadyChan = make(chan struct{})

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

	go func() {
		var subErr error
		if subErr = paStream.Start(); subErr != nil {
			fmt.Println(subErr)
			rootCancel()
		}

	PortThatLoops:
		for {
			select {
			case <-rootCtx.Done():
				break PortThatLoops
			case <-readKickChan:
			}

			if subErr = paStream.Read(); subErr != nil {
				fmt.Println(err)
				rootCancel()
			}

			select {
			case <-rootCtx.Done():
				break PortThatLoops
			case readReadyChan <- struct{}{}:
			}
		}

		if subErr = paStream.Close(); subErr != nil {
			fmt.Println(subErr)
		}

		if subErr = portaudio.Terminate(); subErr != nil {
			fmt.Println(subErr)
		}
	}()

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
