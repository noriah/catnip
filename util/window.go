package util

import "math"

// MovingWindow is a moving window
type MovingWindow struct {
	index    int
	length   int
	capacity int
	variance float64
	stddev   float64
	average  float64

	data []float64
}

func NewMovingWindow(size int) *MovingWindow {
	return &MovingWindow{
		data:     make([]float64, size),
		length:   0,
		capacity: size,
	}
}

func (mw *MovingWindow) calcFinal() (mean float64, stddev float64) {
	if mw.length <= 0 {
		mw.average = 0

	} else if mw.length > 1 {
		// mw.stddev = math.Sqrt(mw.variance / (mw.length - 1))
		// this came from dpayne/cli-visualizer
		stddev = math.Abs((mw.variance / float64(mw.length-1)))
		stddev = math.Sqrt(stddev)
	}

	mw.stddev = stddev

	return mw.Stats()
}

// Update adds the new value to the moving window, returns average and stddev.
// If the window is full, the oldest value will be removed and the new value
// is added. Returns calculated Average and Standard Deviation.
func (mw *MovingWindow) Update(value float64) (mean float64, stddev float64) {
	if mw.length < mw.capacity {
		mw.length++
		mw.average += (value - mw.average) / float64(mw.length)
		mw.variance += math.Pow(value-mw.average, 2.0)

	} else {
		old := mw.data[mw.index]
		newAverage := mw.average + (value-old)/float64(mw.length)
		mw.variance += math.Pow(value-newAverage, 2.0) - math.Pow(old-mw.average, 2.0)
		mw.average = newAverage
	}

	mw.data[mw.index] = value

	if mw.index++; mw.index >= mw.capacity {
		mw.index = 0
	}

	return mw.calcFinal()
}

// Drop removes count values from the window.
func (mw *MovingWindow) Drop(count int) (mean float64, stddev float64) {
	if mw.length <= 0 {
		return mw.calcFinal()
	}

	for count > 0 && mw.length > 0 {
		idx := (mw.index - mw.length)

		if idx < 0 {
			idx = mw.capacity + idx
		}

		old := mw.data[idx]

		mw.variance -= math.Pow(old-mw.average, 2.0)
		mw.average -= old / float64(mw.length)

		mw.length--
		count--
	}

	// If we dont have enough length for standard dev, clear variance
	if mw.length < 2 {
		mw.variance = 0.0

		if mw.length < 1 {
			mw.length = 0
			// same idea with sum. just clear it so we dont have a rouding issue
			mw.average = 0.0
		}
	}

	return mw.calcFinal()
}

func (mw *MovingWindow) Recalculate() (mean float64, stddev float64) {
	if mw.length <= 0 {
		return mw.Stats()
	}

	sum := 0.0

	for count := mw.length; count > 0; count-- {
		idx := (mw.index - count)

		if idx < 0 {
			idx = mw.capacity + idx

		}

		sum += mw.data[idx]
	}

	mw.average = sum / float64(mw.length)

	dev := 0.0

	for count := mw.length; count > 0; count-- {
		idx := (mw.index - count)

		if idx < 0 {
			idx = mw.capacity + idx
		}

		dev += math.Pow(mw.data[idx]-mw.average, 2.0)
	}

	mw.stddev = math.Sqrt(dev / float64(mw.length))

	return mw.Stats()
}

// Len returns how many items in the window
func (mw *MovingWindow) Len() int {
	// logical length
	return mw.length
}

// Cap returns max size of window
func (mw *MovingWindow) Cap() int {
	return mw.capacity
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
func (mw *MovingWindow) Stats() (mean float64, stddev float64) {
	return mw.average, mw.stddev
}
