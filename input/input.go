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
func (sb SampleBuffer) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&sb[0])
}
