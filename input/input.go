package input

import "unsafe"

// Params are input params
type Params struct {
	Device   string  // name of device to look for
	Channels int     // number of channels per frame
	Samples  int     // number of frames per buffer write
	Rate     float64 // sample rate
}

// SampleType is the datatype we want from our inputs
type SampleType = float32

// SampleBuffer is a slice of SampleType
type SampleBuffer []SampleType

// Ptr returns a pointer for use with CGO
func (cb SampleBuffer) Ptr(n ...int) unsafe.Pointer {
	if len(n) > 0 {
		return unsafe.Pointer(&cb[n[0]])

	}

	return unsafe.Pointer(&cb[0])
}
