package util

import "math"

// MovingWindow is a moving window
type MovingWindow struct {
	vals chan float64
	sum  float64
	size float64
	sd   float64
}

// NewMovingWindow returns a new moving window
func NewMovingWindow(size int) *MovingWindow {
	ch := make(chan float64, size)
	for idx := 0; idx < size; idx++ {
		ch <- 0
	}

	return &MovingWindow{
		vals: ch,
		sum:  0,
		size: float64(size),
	}
}

// Update updates the moving window
func (mw *MovingWindow) Update(newVal float64) (float64, float64) {
	val := <-mw.vals
	mw.sd -= math.Pow(val-mw.Mean(), 2)
	mw.sum -= val
	mw.sum += newVal
	mw.sd += math.Pow(newVal-mw.Mean(), 2)
	mw.vals <- newVal
	return mw.Mean(), mw.StdDev()
}

// Mean is the moving window average
func (mw *MovingWindow) Mean() float64 {
	return mw.sum / mw.size
}

// StdDev is the moving average std
func (mw *MovingWindow) StdDev() float64 {
	return math.Sqrt(mw.sd / mw.size)
}
