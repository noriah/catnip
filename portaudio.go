package tavis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gordonklaus/portaudio"
)

// errors
var (
	ErrBadDevice error = errors.New("device not found")
)

type SampleType = float32
type SampleBuffer []SampleType

// Portaudio is an input source that pulls from Portaudio
//
// The number of frames per read will be
type Portaudio struct {
	stream        *portaudio.Stream // our input stream
	scratchBuffer []SampleType      // internal scratch buffer
	loopChanBytes chan int
	loopChanCmplx chan SampleBuffer

	DeviceName string  // name of device to look for
	FrameSize  int     // number of channels per frame
	SampleSize int     // number of frames per buffer write
	SampleRate float64 // sample rate
}

// Init sets up all the portaudio things we need to do
func (pa *Portaudio) Init() error {
	var err error

	if err = portaudio.Initialize(); err != nil {
		return err
	}

	var devices []*portaudio.DeviceInfo

	if devices, err = portaudio.Devices(); err != nil {
		return err
	}

	var device *portaudio.DeviceInfo

	for idx := 0; idx < len(devices); idx++ {
		if strings.Compare(devices[idx].Name, pa.DeviceName) == 0 {
			device = devices[idx]
			break
		}
	}

	if device == nil {
		return ErrBadDevice
	}

	if pa.stream, err = portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device:   device,
				Channels: pa.FrameSize,
				Latency:  device.DefaultLowInputLatency,
			},
			SampleRate:      pa.SampleRate,
			FramesPerBuffer: pa.SampleSize,
			// Flags:           portaudio.ClipOff | portaudio.DitherOff,
		}, pa.PaCallback); err != nil {
		return err
	}

	pa.loopChanCmplx = make(chan SampleBuffer, 1)
	pa.loopChanBytes = make(chan int)

	return err
}

func (pa *Portaudio) Read(ctx context.Context, buf SampleBuffer) int {
	select {
	case pa.loopChanCmplx <- buf:
		select {
		case dst := <-pa.loopChanBytes:
			return dst

		case <-ctx.Done():
			fmt.Println("ctx.Done signal - on return")
		}

	case <-ctx.Done():
		fmt.Println("ctx.Done signal - on recieve")
	}

	return 0
}

// Close closes the close close
func (pa *Portaudio) Close() error {
	return pa.stream.Close()
}

// PaCallback does a call to the back for stock check
func (pa *Portaudio) PaCallback(in SampleBuffer,
	timeInfo portaudio.StreamCallbackTimeInfo,
	flags portaudio.StreamCallbackFlags) {

	select {
	case dst := <-pa.loopChanCmplx:

		var idx int
		for idx = 0; idx < len(in) && idx < len(dst); idx++ {
			dst[idx] = in[idx]
		}

		select {
		case pa.loopChanBytes <- idx:
		case <-time.After(time.Second):
			fmt.Println("error?? top")
		}
	default:
	}
}

// Start will open the stream and start audio processing
func (pa *Portaudio) Start() {
	pa.stream.Start()
}

// Stop does the stop
func (pa *Portaudio) Stop() error {
	return pa.stream.Stop()
}
