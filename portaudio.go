package tavis

import (
	"context"
	"errors"
	"log"

	"github.com/noriah/tavis/portaudio"
)

// errors
var (
	ErrBadDevice    error = errors.New("device not found")
	ErrReadTimedOut error = errors.New("read timed out")
)

// SampleType is broken out because portaudio supports different types
type SampleType = float32

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

	if err := portaudio.Initialize(); err != nil {
		return err
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return err
	}

	var device *portaudio.DeviceInfo

	for _, d := range devices {
		if d.Name == pa.DeviceName {
			device = d
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
	for pa.ReadyRead() < pa.SampleSize*pa.FrameSize {
		select {
		case <-ctx.Done():
			log.Println("read timed out")
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
