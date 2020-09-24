package input

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gordonklaus/portaudio"
	"github.com/noriah/tavis"
)

// errors
var (
	ErrBadDevice error = errors.New("device not found")
)

// Portaudio is an input source that pulls from Portaudio
//
// The number of frames per read will be
type Portaudio struct {
	DeviceName   string             // name of device to look for
	FrameSize    int                // number of channels per frame
	SampleSize   int                // number of frames per buffer write
	SampleRate   float64            // sample rate
	SampleBuffer []tavis.SampleType // slice pointing to our dest buffer
	stream       *portaudio.Stream  // our input stream
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
			Flags:           portaudio.ClipOff | portaudio.DitherOff,
		}, pa.SampleBuffer); err != nil {
		return err
	}

	return err
}

func (pa *Portaudio) paCallback() {

}

// Start will open the stream and start audio processing
func (pa *Portaudio) Start(rootCtx context.Context) context.CancelFunc {
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
				break PortThatLoops
			}

			select {
			case <-rootCtx.Done():
				break PortThatLoops
			case readReadyChan <- true:
			}
		}

		if subErr = paStream.Close(); subErr != nil {
			fmt.Println(subErr)
		}

		if subErr = portaudio.Terminate(); subErr != nil {
			fmt.Println(subErr)
		}
	}()

}
