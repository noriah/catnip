package input

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/noriah/tavis/input/portaudio"
)

// errors
var (
	ErrBadDevice    error = errors.New("device not found")
	ErrReadTimedOut error = errors.New("read timed out")
)

// Portaudio is an input source that pulls from Portaudio
//
// The number of frames per read will be
type Portaudio struct {
	stream *portaudio.Stream // our input stream

	sampleBuffer SampleBuffer // internal scratch buffer

	deviceName string  // name of device to look for
	frameSize  int     // number of channels per frame
	sampleSize int     // number of frames per buffer write
	sampleRate float64 // sample rate
}

// NewPortaudio returns a new portaudio input
func NewPortaudio(pref Params) *Portaudio {

	var newBuf SampleBuffer = make(SampleBuffer, pref.Samples*pref.Channels)

	var pa *Portaudio = &Portaudio{
		sampleBuffer: newBuf,
		deviceName:   pref.Device,
		frameSize:    pref.Channels,
		sampleSize:   pref.Samples,
		sampleRate:   pref.Rate,
	}

	if err := pa.init(); err != nil {
		panic(err)
	}

	return pa
}

// Init sets up all the portaudio things we need to do
func (pa *Portaudio) init() error {
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
		if strings.Compare(devices[idx].Name, pa.deviceName) == 0 {
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
				Channels: pa.frameSize,
				Latency:  device.DefaultLowInputLatency,
			},
			SampleRate:      pa.sampleRate,
			FramesPerBuffer: pa.sampleSize,
			Flags:           portaudio.ClipOff | portaudio.DitherOff,
		}, pa.sampleBuffer); err != nil {
		return err
	}

	return err
}

// Buffer returns a slice to our buffer
func (pa *Portaudio) Buffer() SampleBuffer {
	return pa.sampleBuffer[:]
}

// ReadyRead returns the number of frames ready to read
func (pa *Portaudio) ReadyRead() int {
	var ready, _ = pa.stream.AvailableToRead()
	return ready
}

// Read signals portaudio to dump some data into the buffer we gave it.
// Will block if there is not enough data yet.
func (pa *Portaudio) Read(ctx context.Context) error {
	for pa.ReadyRead() < pa.sampleSize {
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
