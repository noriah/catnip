package util

import "math"

// MovingWindow is a moving window
type MovingWindow struct {
	vals chan float64
	sum  float64
	size float64
	cap  float64
	sd   float64
}

// NewMovingWindow returns a new moving window
func NewMovingWindow(size int) *MovingWindow {

	return &MovingWindow{
		vals: make(chan float64, size),
		sum:  0,
		size: 0,
		cap:  float64(size),
	}
}

// Update updates the moving window
// The "hack" with this standard deviation is false and wrong
// But idc. im just poking numbers right now.
// I will do something proper later
func (mw *MovingWindow) Update(newVal float64) (float64, float64) {
	if mw.size >= mw.cap {
		val := <-mw.vals
		mw.sum -= val
		mw.size--
		mw.sd -= math.Pow(val-mw.Mean(), 2)
	}
	mw.size++
	mw.sum += newVal
	mw.sd += math.Pow(newVal-mw.Mean(), 2)
	mw.vals <- newVal
	return mw.Mean(), mw.StdDev()
}

// Drop removes count items from the window
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	var val float64
	for cnt := 0; cnt < count && mw.size > 0; cnt++ {
		val = <-mw.vals
		mw.sum -= val
		mw.size--
		mw.sd -= math.Pow(val-mw.Mean(), 2)
	}

	return mw.Mean(), mw.StdDev()
}

// Size returns how many items in the window
func (mw *MovingWindow) Size() int {
	return int(mw.size)
}

// Mean is the moving window average
func (mw *MovingWindow) Mean() float64 {
	if mw.size == 0 {
		return 0
	}

	return mw.sum / mw.size
}

// StdDev is the moving average std
func (mw *MovingWindow) StdDev() float64 {
	if mw.size == 0 {
		return 0
	}

	return math.Sqrt(mw.sd / mw.size)
}
