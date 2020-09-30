package tavis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/noriah/tavis/portaudio"
)

// errors
var (
	ErrBadDevice    error = errors.New("device not found")
	ErrReadTimedOut error = errors.New("read timed out")
)

// SampleType is broken out because portaudio supports different types
type SampleType = float32

// Params are input params
type Params struct {
	Device   string  // name of device to look for
	Channels int     // number of channels per frame
	Samples  int     // number of frames per buffer write
	Rate     float64 // sample rate
}

// Portaudio is an input source that pulls from Portaudio
//
// The number of frames per read will be
type Portaudio struct {
	stream *portaudio.Stream // our input stream

	sampleBuffer []SampleType // internal scratch buffer

	DeviceName string  // name of device to look for
	FrameSize  int     // number of channels per frame
	SampleSize int     // number of frames per buffer write
	SampleRate float64 // sample rate
}

// Init sets up all the portaudio things we need to do
func (pa *Portaudio) Init() error {

	pa.sampleBuffer = make([]SampleType, pa.SampleSize*pa.FrameSize)

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
		}, &pa.sampleBuffer); err != nil {
		return err
	}

	return err
}

// Buffer returns a slice to our buffer
func (pa *Portaudio) Buffer() []SampleType {
	return pa.sampleBuffer
}

// ReadyRead returns the number of frames ready to read
func (pa *Portaudio) ReadyRead() int {
	var ready, _ = pa.stream.AvailableToRead()
	return ready
}

// Read signals portaudio to dump some data into the buffer we gave it.
// Will block if there is not enough data yet.
func (pa *Portaudio) Read(ctx context.Context) error {
	for pa.ReadyRead() < pa.SampleSize {
		select {
		case <-ctx.Done():
			fmt.Println("read timed out")
			return ErrReadTimedOut
		default:
		}
	}

	return pa.stream.Read()
}

// Close closes the close close
func (pa *Portaudio) Close() error {
	defer portaudio.Terminate()
	return pa.stream.Close()
}

// Start will open the stream and start audio processing
func (pa *Portaudio) Start() {
	pa.stream.Start()
}

// Stop does the stop
func (pa *Portaudio) Stop() error {
	return pa.stream.Stop()
}
