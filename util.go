package tavis

import (
	"math"
)

// MovingWindow is a moving window
type MovingWindow struct {
	variance float64
	stddev   float64
	sum      float64
	average  float64
	points   float64
	capacity float64
	window   chan float64
}

// NewMovingWindow returns a new moving window
func NewMovingWindow(points int) *MovingWindow {

	return &MovingWindow{
		window:   make(chan float64, points),
		capacity: float64(points),
	}
}

func (mw *MovingWindow) calcRaw(new, old float64) {
	mw.variance = mw.variance + (new * new) - (old * old)
	mw.sum = mw.sum + (new - old)
}

func (mw *MovingWindow) calcFinal() (float64, float64) {
	if mw.points > 0 {
		mw.average = mw.sum / mw.points

		if mw.points > 1 {
			// mw.stddev = math.Sqrt(mw.variance / (mw.points - 1))
			// okay so this came from dpayne/cli-visualizer
			mw.stddev = (mw.variance / mw.points) - math.Pow(mw.average, 2)
			mw.stddev = math.Sqrt(mw.stddev)
		} else {
			mw.stddev = 0
		}
	} else {
		mw.average = 0
		mw.stddev = 0
	}

	return mw.average, mw.stddev
}

// Update updates the moving window
// If the moving window is at capacity, pop the oldest, and push value
func (mw *MovingWindow) Update(value float64) (float64, float64) {
	if mw.points < mw.capacity {
		mw.calcRaw(value, 0)
		mw.points++
	} else {
		mw.calcRaw(value, <-mw.window)
	}

	mw.window <- value
	return mw.calcFinal()
}

// Drop removes count items from the window
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	for ; count > 0 && mw.points > 0; count-- {
		mw.points--
		mw.calcRaw(0, <-mw.window)
	}

	// If we dont have enough points for standard dev, clear variance
	if mw.points < 2 {
		mw.variance = 0
		if mw.points < 1 {
			// mw.points = 0
			// same idea with sum. just clear it so we dont have a rouding issue
			mw.sum = 0
		}
	}

	return mw.calcFinal()
}

// Points returns how many items in the window
func (mw *MovingWindow) Points() int {
	return int(mw.points)
}

// Capacity returns max size of window
func (mw *MovingWindow) Capacity() int {
	return int(mw.capacity)
}

// Mean is the moving window average
func (mw *MovingWindow) Mean() float64 {
	return mw.average
}

// StdDev is the moving average std
func (mw *MovingWindow) StdDev() float64 {
	return mw.stddev
}

// Stats returns the statistics of this window
func (mw *MovingWindow) Stats() (float64, float64) {
	return mw.average, mw.stddev
}
