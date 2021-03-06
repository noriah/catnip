// Package portaudio contains modifications of CGO bindings for portaudio
//
// obtained from https://github.com/gordonklaus/portaudio
// see ./LICENSE
package portaudio

// #cgo pkg-config: portaudio-2.0
// #include <portaudio.h>
import "C"

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/noriah/catnip/input/common/endian"
)

// Version returns the release number of PortAudio.
func Version() int {
	return int(C.Pa_GetVersion())
}

// VersionText returns the textual description of the PortAudio release.
func VersionText() string {
	return C.GoString(C.Pa_GetVersionText())
}

// Error wraps over PaError.
type Error C.PaError

func (err Error) Error() string {
	return C.GoString(C.Pa_GetErrorText(C.PaError(err)))
}

// PortAudio Errors.
const (
	NotInitialized                        Error = C.paNotInitialized
	InvalidChannelCount                   Error = C.paInvalidChannelCount
	InvalidSampleRate                     Error = C.paInvalidSampleRate
	InvalidDevice                         Error = C.paInvalidDevice
	InvalidFlag                           Error = C.paInvalidFlag
	SampleFormatNotSupported              Error = C.paSampleFormatNotSupported
	BadIODeviceCombination                Error = C.paBadIODeviceCombination
	InsufficientMemory                    Error = C.paInsufficientMemory
	BufferTooBig                          Error = C.paBufferTooBig
	BufferTooSmall                        Error = C.paBufferTooSmall
	NullCallback                          Error = C.paNullCallback
	BadStreamPtr                          Error = C.paBadStreamPtr
	TimedOut                              Error = C.paTimedOut
	InternalError                         Error = C.paInternalError
	DeviceUnavailable                     Error = C.paDeviceUnavailable
	IncompatibleHostApiSpecificStreamInfo Error = C.paIncompatibleHostApiSpecificStreamInfo
	StreamIsStopped                       Error = C.paStreamIsStopped
	StreamIsNotStopped                    Error = C.paStreamIsNotStopped
	InputOverflowed                       Error = C.paInputOverflowed
	OutputUnderflowed                     Error = C.paOutputUnderflowed
	HostApiNotFound                       Error = C.paHostApiNotFound
	InvalidHostApi                        Error = C.paInvalidHostApi
	IncompatibleStreamHostApi             Error = C.paIncompatibleStreamHostApi
	BadBufferPtr                          Error = C.paBadBufferPtr
)

// UnanticipatedHostError contains details for ApiHost related errors.
type UnanticipatedHostError struct {
	HostApiType HostApiType
	Code        int
	Text        string
}

func (err UnanticipatedHostError) Error() string {
	return err.Text
}

func newError(err C.PaError) error {
	switch err {
	case C.paUnanticipatedHostError:
		hostErr := C.Pa_GetLastHostErrorInfo()
		return UnanticipatedHostError{
			HostApiType(hostErr.hostApiType),
			int(hostErr.errorCode),
			C.GoString(hostErr.errorText),
		}
	case C.paNoError:
		return nil
	}
	return Error(err)
}

var initialized = 0

// Initialize initializes internal data structures and
// prepares underlying host APIs for use. With the exception
// of Version(), VersionText(), and ErrorText(), this function
// MUST be called before using any other PortAudio API functions.
//
// If Initialize() is called multiple times, each successful call
// must be matched with a corresponding call to Terminate(). Pairs of
// calls to Initialize()/Terminate() may overlap, and are not required to be fully nested.
//
// Note that if Initialize() returns an error code, Terminate() should NOT be called.
func Initialize() error {
	paErr := C.Pa_Initialize()
	if paErr != C.paNoError {
		return newError(paErr)
	}
	return nil
}

// Terminate deallocates all resources allocated by PortAudio
// since it was initialized by a call to Initialize().
//
// In cases where Initialize() has been called multiple times,
// each call must be matched with a corresponding call to Pa_Terminate().
// The final matching call to Pa_Terminate() will automatically
// close any PortAudio streams that are still open..
//
// Terminate MUST be called before exiting a program which uses PortAudio.
// Failure to do so may result in serious resource leaks, such as audio devices
// not being available until the next reboot.
func Terminate() error {
	paErr := C.Pa_Terminate()
	if paErr != C.paNoError {
		return newError(paErr)
	}
	return nil
}

// HostApiType maps ints to HostApi modes.
type HostApiType int

func (t HostApiType) String() string {
	return hostApiStrings[t]
}

var hostApiStrings = [...]string{
	InDevelopment:   "InDevelopment",
	DirectSound:     "DirectSound",
	MME:             "MME",
	ASIO:            "ASIO",
	SoundManager:    "SoundManager",
	CoreAudio:       "CoreAudio",
	OSS:             "OSS",
	ALSA:            "ALSA",
	AL:              "AL",
	BeOS:            "BeOS",
	WDMkS:           "WDMKS",
	JACK:            "JACK",
	WASAPI:          "WASAPI",
	AudioScienceHPI: "AudioScienceHPI",
}

// PortAudio Api types.
const (
	InDevelopment   HostApiType = C.paInDevelopment
	DirectSound     HostApiType = C.paDirectSound
	MME             HostApiType = C.paMME
	ASIO            HostApiType = C.paASIO
	SoundManager    HostApiType = C.paSoundManager
	CoreAudio       HostApiType = C.paCoreAudio
	OSS             HostApiType = C.paOSS
	ALSA            HostApiType = C.paALSA
	AL              HostApiType = C.paAL
	BeOS            HostApiType = C.paBeOS
	WDMkS           HostApiType = C.paWDMKS
	JACK            HostApiType = C.paJACK
	WASAPI          HostApiType = C.paWASAPI
	AudioScienceHPI HostApiType = C.paAudioScienceHPI
)

// HostApiInfo contains information for a HostApi.
type HostApiInfo struct {
	Type                HostApiType
	Name                string
	DefaultInputDevice  *DeviceInfo
	DefaultOutputDevice *DeviceInfo
	Devices             []*DeviceInfo
}

// DeviceInfo contains information for an audio device.
type DeviceInfo struct {
	index                    C.PaDeviceIndex
	Name                     string
	MaxInputChannels         int
	MaxOutputChannels        int
	DefaultLowInputLatency   time.Duration
	DefaultLowOutputLatency  time.Duration
	DefaultHighInputLatency  time.Duration
	DefaultHighOutputLatency time.Duration
	DefaultSampleRate        float64
	HostApi                  *HostApiInfo
}

// HostApis returns all information available for HostApis.
func HostApis() ([]*HostApiInfo, error) {
	hosts, _, err := hostsAndDevices()
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// HostApi returns information for a requested HostApiType.
func HostApi(apiType HostApiType) (*HostApiInfo, error) {
	hosts, err := HostApis()
	if err != nil {
		return nil, err
	}
	i := C.Pa_HostApiTypeIdToHostApiIndex(C.PaHostApiTypeId(apiType))
	if i < 0 {
		return nil, newError(C.PaError(i))
	}
	return hosts[i], nil
}

// DefaultHostApi returns information of the default HostApi available on the system.
//
// The default host API will be the lowest common denominator host API
// on the current platform and is unlikely to provide the best performance.
func DefaultHostApi() (*HostApiInfo, error) {
	hosts, err := HostApis()
	if err != nil {
		return nil, err
	}
	i := C.Pa_GetDefaultHostApi()
	if i < 0 {
		return nil, newError(C.PaError(i))
	}
	return hosts[i], nil
}

// Devices returns information for all available devices on the system.
func Devices() ([]*DeviceInfo, error) {
	_, devs, err := hostsAndDevices()
	if err != nil {
		return nil, err
	}
	return devs, nil
}

/*
Cache the HostApi/Device list to simplify the enumeration code.
Note that portaudio itself caches the lists, so these won't go stale.

However, there is talk of extending the portaudio API to allow clients
to rescan available devices without calling Pa_Terminate() followed by
Pa_Initialize() - our caching strategy will have to change if this
goes ahead. See https://www.assembla.com/spaces/portaudio/tickets/11
*/
var (
	cached   bool
	hostApis []*HostApiInfo
	devices  []*DeviceInfo
)

func hostsAndDevices() ([]*HostApiInfo, []*DeviceInfo, error) {
	if !cached {
		nhosts := C.Pa_GetHostApiCount()
		ndevs := C.Pa_GetDeviceCount()
		if nhosts < 0 {
			return nil, nil, newError(C.PaError(nhosts))
		}
		if ndevs < 0 {
			return nil, nil, newError(C.PaError(ndevs))
		}
		devices = make([]*DeviceInfo, ndevs)
		hosti := make([]C.PaHostApiIndex, ndevs)
		for i := range devices {
			i := C.PaDeviceIndex(i)
			paDev := C.Pa_GetDeviceInfo(i)
			devices[i] = &DeviceInfo{
				index:                    i,
				Name:                     C.GoString(paDev.name),
				MaxInputChannels:         int(paDev.maxInputChannels),
				MaxOutputChannels:        int(paDev.maxOutputChannels),
				DefaultLowInputLatency:   duration(paDev.defaultLowInputLatency),
				DefaultLowOutputLatency:  duration(paDev.defaultLowOutputLatency),
				DefaultHighInputLatency:  duration(paDev.defaultHighInputLatency),
				DefaultHighOutputLatency: duration(paDev.defaultHighOutputLatency),
				DefaultSampleRate:        float64(paDev.defaultSampleRate),
			}
			hosti[i] = paDev.hostApi
		}
		hostApis = make([]*HostApiInfo, nhosts)
		for i := range hostApis {
			i := C.PaHostApiIndex(i)
			paHost := C.Pa_GetHostApiInfo(i)
			devs := make([]*DeviceInfo, paHost.deviceCount)
			for j := range devs {
				devs[j] = devices[C.Pa_HostApiDeviceIndexToDeviceIndex(i, C.int(j))]
			}
			hostApis[i] = &HostApiInfo{
				Type:                HostApiType(paHost._type),
				Name:                C.GoString(paHost.name),
				DefaultInputDevice:  lookupDevice(devices, paHost.defaultInputDevice),
				DefaultOutputDevice: lookupDevice(devices, paHost.defaultOutputDevice),
				Devices:             devs,
			}
		}
		for i := range devices {
			devices[i].HostApi = hostApis[hosti[i]]
		}
		cached = true
	}
	return hostApis, devices, nil
}

func duration(paTime C.PaTime) time.Duration {
	return time.Duration(paTime * C.PaTime(time.Second))
}

func lookupDevice(d []*DeviceInfo, i C.PaDeviceIndex) *DeviceInfo {
	if i >= 0 {
		return d[i]
	}
	return nil
}

// StreamParameters includes all parameters required to
// open a stream except for the callback or buffers.
type StreamParameters struct {
	Input, Output   StreamDeviceParameters
	SampleRate      float64
	FramesPerBuffer int
	Flags           StreamFlags
}

// StreamDeviceParameters specifies parameters for
// one device (either input or output) in a stream.
// A nil Device indicates that no device is to be used
// -- i.e., for an input- or output-only stream.
type StreamDeviceParameters struct {
	Device   *DeviceInfo
	Channels int
	Latency  time.Duration
}

// FramesPerBufferUnspecified ...
const FramesPerBufferUnspecified = C.paFramesPerBufferUnspecified

// StreamFlags ...
type StreamFlags C.PaStreamFlags

// Flags for Streams
const (
	NoFlag                                StreamFlags = C.paNoFlag
	ClipOff                               StreamFlags = C.paClipOff
	DitherOff                             StreamFlags = C.paDitherOff
	NeverDropInput                        StreamFlags = C.paNeverDropInput
	PrimeOutputBuffersUsingStreamCallback StreamFlags = C.paPrimeOutputBuffersUsingStreamCallback
	PlatformSpecificFlags                 StreamFlags = C.paPlatformSpecificFlags
)

// IsFormatSupported Returns nil if the format is supported, otherwise an error.
// The args parameter has the same meaning as in OpenStream.
func IsFormatSupported(p StreamParameters, args ...interface{}) error {
	s := &Stream{}
	err := s.init(p, args...)
	if err != nil {
		return err
	}
	return newError(C.Pa_IsFormatSupported(s.inParams, s.outParams, C.double(p.SampleRate)))
}

// Int24 holds the bytes of a 24-bit signed integer in native byte order.
type Int24 [3]byte

// PutInt32 puts the three most significant bytes of i32 into i24.
func (i24 *Int24) PutInt32(i32 int32) {
	if littleEndian {
		i24[0] = byte(i32 >> 8)
		i24[1] = byte(i32 >> 16)
		i24[2] = byte(i32 >> 24)
	} else {
		i24[2] = byte(i32 >> 8)
		i24[1] = byte(i32 >> 16)
		i24[0] = byte(i32 >> 24)
	}
}

var littleEndian = endian.IsLE()

// Stream provides access to audio hardware represented
// by one or more PaDevices. Depending on the underlying
// Host API, it may be possible to open multiple streams
// using the same device, however this behavior is
// implementation defined.
//
// Portable applications should assume that a Device may be simultaneously used by at most one Stream.
type Stream struct {
	id                  uintptr
	paStream            unsafe.Pointer
	inParams, outParams *C.PaStreamParameters
	in, out             *reflect.SliceHeader
	args                []reflect.Value
	callback            reflect.Value
	closed              bool
}

/*
Since Go 1.6, if a Go pointer is passed to C then the Go memory it points to
may not contain any Go pointers: https://golang.org/cmd/cgo/#hdr-Passing_pointers
To deal with this, we maintain an id-keyed map of active streams.
*/
var (
	mu      sync.RWMutex
	streams = map[uintptr]*Stream{}
	nextID  uintptr
)

func newStream() *Stream {
	mu.Lock()
	defer mu.Unlock()
	s := &Stream{id: nextID}
	streams[nextID] = s
	nextID++
	return s
}

func getStream(id uintptr) *Stream {
	mu.RLock()
	defer mu.RUnlock()
	return streams[id]
}

func delStream(s *Stream) {
	mu.Lock()
	defer mu.Unlock()
	delete(streams, s.id)
}

// OpenStream creates an instance of a Stream.
//
// For an input- or output-only stream, p.Output.Device or p.Input.Device must be nil, respectively.
//
// The args may consist of either a single StreamCallback or,
// for a blocking stream, two Buffers or pointers to Buffers.
//
// For an input- or output-only stream, one of the Buffer args may be omitted.
func OpenStream(p StreamParameters, args ...interface{}) (*Stream, error) {
	s := newStream()
	err := s.init(p, args...)
	if err != nil {
		delStream(s)
		return nil, err
	}
	paErr := C.Pa_OpenStream(&s.paStream, s.inParams, nil, C.double(p.SampleRate), C.ulong(p.FramesPerBuffer), C.PaStreamFlags(p.Flags), nil, unsafe.Pointer(s.id))
	if paErr != C.paNoError {
		delStream(s)
		return nil, newError(paErr)
	}
	return s, nil
}

func (s *Stream) init(p StreamParameters, args ...interface{}) error {
	switch len(args) {
	case 0:
		return fmt.Errorf("too few args")
	case 1, 2:
		return s.initBuffers(p, args...)
	default:
		return fmt.Errorf("too many args")
	}
}

func (s *Stream) initBuffers(p StreamParameters, args ...interface{}) error {
	bothBufs := len(args) == 2
	bufArg := func(p StreamDeviceParameters) (*C.PaStreamParameters, *reflect.SliceHeader, error) {
		if p.Device != nil || bothBufs {
			if len(args) == 0 {
				return nil, nil, fmt.Errorf("too few Buffer args")
			}
			arg := reflect.ValueOf(args[0])
			args = args[1:]
			t := arg.Type()
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			} else {
				argPtr := reflect.New(t)
				argPtr.Elem().Set(arg)
				arg = argPtr
			}
			sampleFmt := sampleFormat(t)
			if sampleFmt == 0 {
				return nil, nil, fmt.Errorf("invalid Buffer type %v", t)
			}
			if arg.IsNil() {
				return nil, nil, fmt.Errorf("nil Buffer pointer")
			}
			if p.Device != nil {
				return paStreamParameters(p, sampleFmt), (*reflect.SliceHeader)(unsafe.Pointer(arg.Pointer())), nil
			}
		}
		return nil, nil, nil
	}
	var err error
	s.inParams, s.in, err = bufArg(p.Input)
	if err != nil {
		return err
	}
	s.outParams, s.out, err = bufArg(p.Output)
	if err != nil {
		return err
	}
	return nil
}

// FormatFlag is a port format
type FormatFlag int

// Port format constants
const (
	NonInterleaved FormatFlag = C.paNonInterleaved
	Float32        FormatFlag = C.paFloat32
	Int32          FormatFlag = C.paInt32
	Int16          FormatFlag = C.paInt16
	Int8           FormatFlag = C.paInt8
	Uint8          FormatFlag = C.paUInt8
)

func sampleFormat(b reflect.Type) (f C.PaSampleFormat) {
	if b.Kind() != reflect.Slice {
		return 0
	}
	b = b.Elem()
	if b.Kind() == reflect.Slice {
		f = C.paNonInterleaved
		b = b.Elem()
	}
	switch b.Kind() {
	case reflect.Float32:
		f |= C.paFloat32
	case reflect.Int32:
		f |= C.paInt32
	default:
		if b == reflect.TypeOf(Int24{}) {
			f |= C.paInt24
		} else {
			return 0
		}
	case reflect.Int16:
		f |= C.paInt16
	case reflect.Int8:
		f |= C.paInt8
	case reflect.Uint8:
		f |= C.paUInt8
	}
	return f
}

func paStreamParameters(p StreamDeviceParameters, fmt C.PaSampleFormat) *C.PaStreamParameters {
	return &C.PaStreamParameters{
		device:           p.Device.index,
		channelCount:     C.int(p.Channels),
		sampleFormat:     fmt,
		suggestedLatency: C.PaTime(p.Latency.Seconds()),
	}
}

// Close terminates the stream.
func (s *Stream) Close() error {
	if !s.closed {
		s.closed = true
		err := newError(C.Pa_CloseStream(s.paStream))
		delStream(s)
		return err
	}
	return nil
}

// Start commences audio processing.
func (s *Stream) Start() error {
	return newError(C.Pa_StartStream(s.paStream))
}

// Stop terminates audio processing. It waits until all pending
// audio buffers have been played before it returns.
func (s *Stream) Stop() error {
	return newError(C.Pa_StopStream(s.paStream))
}

// Abort terminates audio processing immediately
// without waiting for pending buffers to complete.
func (s *Stream) Abort() error {
	return newError(C.Pa_AbortStream(s.paStream))
}

// Info returns information about the Stream instance.
func (s *Stream) Info() *StreamInfo {
	i := C.Pa_GetStreamInfo(s.paStream)
	if i == nil {
		return nil
	}
	return &StreamInfo{duration(i.inputLatency), duration(i.outputLatency), float64(i.sampleRate)}
}

// StreamInfo contains information about the stream.
type StreamInfo struct {
	InputLatency, OutputLatency time.Duration
	SampleRate                  float64
}

// Time returns the current time in seconds for a lifespan of a stream.
// Starting and stopping the stream does not affect the passage of time.
func (s *Stream) Time() time.Duration {
	return duration(C.Pa_GetStreamTime(s.paStream))
}

// CpuLoad returns the CPU usage information for the specified stream,
// where 0.0 is 0% usage and 1.0 is 100% usage.
//
// The "CPU Load" is a fraction of total CPU time consumed by a
// callback stream's audio processing routines including,
// but not limited to the client supplied stream callback.
//
// This function does not work with blocking read/write streams.
//
// This function may be called from the stream callback function or the application.
func (s *Stream) CpuLoad() float64 {
	return float64(C.Pa_GetStreamCpuLoad(s.paStream))
}

// AvailableToRead returns the number of frames that
// can be read from the stream without waiting.
func (s *Stream) AvailableToRead() (int, error) {
	n := C.Pa_GetStreamReadAvailable(s.paStream)
	if n < 0 {
		return 0, newError(C.PaError(n))
	}
	return int(n), nil
}

// Read uses the buffer provided to OpenStream.
// The number of samples to read is determined by the size of the buffer.
func (s *Stream) Read() error {
	buf, frames, err := getBuffer(s.in, s.inParams)
	if err != nil {
		return err
	}
	return newError(C.Pa_ReadStream(s.paStream, buf, C.ulong(frames)))
}

func getBuffer(s *reflect.SliceHeader, p *C.PaStreamParameters) (unsafe.Pointer, int, error) {
	if p.sampleFormat&C.paNonInterleaved == 0 {
		n := int(p.channelCount)
		if s.Len%n != 0 {
			return nil, 0, fmt.Errorf("length of interleaved buffer not divisible by number of channels")
		}
		return unsafe.Pointer(s.Data), s.Len / n, nil
	} else {
		if s.Len != int(p.channelCount) {
			return nil, 0, fmt.Errorf("buffer has wrong number of channels")
		}
		buf := make([]uintptr, s.Len)
		frames := -1
		sp := s.Data
		for i := range buf {
			ch := (*reflect.SliceHeader)(unsafe.Pointer(sp))
			if frames == -1 {
				frames = ch.Len
			} else if ch.Len != frames {
				return nil, 0, fmt.Errorf("channels have different lengths")
			}
			buf[i] = ch.Data
			sp += unsafe.Sizeof(reflect.SliceHeader{})
		}
		return unsafe.Pointer(&buf[0]), frames, nil
	}
}
