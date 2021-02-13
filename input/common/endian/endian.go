package endian

import "unsafe"

// IsLE returns true if the host architecture is little-endian.
func IsLE() bool {
	x := 1
	return *(*byte)(unsafe.Pointer(&x)) == 1
}
