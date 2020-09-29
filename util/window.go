package util

import (
	"fmt"
	"math"
)

// MovingWindow is a moving window
type MovingWindow struct {
	vr   float64
	sd   float64
	sum  float64
	mean float64
	size float64
	cap  float64
	vals chan float64
}

// NewMovingWindow returns a new moving window
func NewMovingWindow(size int) *MovingWindow {

	return &MovingWindow{
		vals: make(chan float64, size),
		cap:  float64(size),
	}
}

// Update updates the moving window
// The "hack" with this standard deviation is false and wrong
// But idc. im just poking numbers right now.
// I will do something proper later
func (mw *MovingWindow) Update(val float64) (float64, float64) {
	if mw.size >= mw.cap {
		mw.pushpop(val, <-mw.vals)
		mw.vals <- val
		return mw.mean, mw.sd
	}

	mw.size++
	mw.vals <- val
	mw.pushpop(val, 0)
	return mw.mean, mw.sd
}

func (mw *MovingWindow) pushpop(new, old float64) {
	mw.vr = mw.vr + (new * new) - (old * old)
	mw.sum = mw.sum + (new - old)
	mw.mean = mw.sum / mw.size
	mw.sd = math.Sqrt(mw.vr / (mw.size - 1))

	fmt.Println(new, old, mw.mean, mw.sd, mw.vr)
}

// Drop removes count items from the window
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	for cnt := 0; cnt < count; cnt++ {

		mw.size--
		// if we emptied out the window, set to 0 and return
		if mw.size <= 0 {
			mw.vr = 0
			mw.sd = 0
			mw.sum = 0
			mw.mean = 0
			mw.size = 0
			// Get the last element out
			<-mw.vals
			break
		}
		mw.pushpop(0, <-mw.vals)
	}

	return mw.mean, mw.sd
}

// Size returns how many items in the window
func (mw *MovingWindow) Size() int {
	return int(mw.size)
}

// Mean is the moving window average
func (mw *MovingWindow) Mean() float64 {
	return mw.mean
}

// StdDev is the moving average std
func (mw *MovingWindow) StdDev() float64 {
	return mw.sd
}
