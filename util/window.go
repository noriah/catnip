package util

import (
	"math"
)

// MovingWindow is a moving window
type MovingWindow struct {
	index  int
	length int

	variance float64
	stddev   float64

	sum     float64
	average float64

	Capacity int

	Data []float64
}

func (mw *MovingWindow) calcFinal() (float64, float64) {
	if mw.length > 1 {
		// mw.stddev = math.Sqrt(mw.variance / (mw.length - 1))
		// okay so this came from dpayne/cli-visualizer
		mw.stddev = (mw.variance / float64(mw.length-1)) - (mw.average * mw.average)
		if mw.stddev < 0.0 {
			mw.stddev = -mw.stddev
		}
		mw.stddev = math.Sqrt(mw.stddev)
	} else {
		mw.stddev = 0
	}

	if mw.length > 0 {
		mw.average = mw.sum / float64(mw.length)
	} else {
		mw.average = 0
	}

	return mw.average, mw.stddev
}

// Update updates the moving window
func (mw *MovingWindow) Update(value float64) (float64, float64) {
	if mw.length < mw.Capacity {

		mw.length++

		mw.sum += value
		mw.variance += (value * value)

	} else {
		var old = mw.Data[mw.index]
		mw.sum += value - old
		mw.variance += (value * value) - (old * old)
	}

	mw.Data[mw.index] = value

	if mw.index++; mw.index >= mw.Capacity {
		mw.index = 0
	}

	return mw.calcFinal()
}

// Drop removes count items from the window
// TODO(winter): look into a better index calculation
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	if mw.length <= 0 {
		return mw.calcFinal()
	}

	for count > 0 && mw.length > 0 {

		var idx = (mw.index - mw.length)
		if idx < 0 {
			idx = mw.Capacity + idx
		}

		var old = mw.Data[idx]

		mw.sum -= old
		mw.variance -= old * old

		mw.length--

		count--
	}

	// If we dont have enough length for standard dev, clear variance
	if mw.length < 2 {
		mw.variance = 0
		if mw.length < 1 {
			mw.length = 0
			// same idea with sum. just clear it so we dont have a rouding issue
			mw.sum = 0
		}
	}

	return mw.calcFinal()
}

// Len returns how many items in the window
func (mw *MovingWindow) Len() int {
	// logical length
	return mw.length
}

// Cap returns max size of window
func (mw *MovingWindow) Cap() int {
	return mw.Capacity
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
