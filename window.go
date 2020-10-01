package tavis

import (
	"math"
)

type node struct {
	next  *node
	value float64
}

// MovingWindow is a moving window
type MovingWindow struct {
	root *node
	tail *node

	length   int
	capacity int

	variance float64
	stddev   float64

	sum     float64
	average float64
}

// NewMovingWindow returns a new moving window.
func NewMovingWindow(size int) *MovingWindow {

	var mw = &MovingWindow{
		root:     &node{},
		capacity: size,
	}

	mw.root.next = mw.root
	mw.tail = mw.root

	return mw
}

// TODO(noriah): resource pool for nodes would be nice

func (mw *MovingWindow) enq(value float64) {

	mw.tail.next = &node{
		next:  mw.root,
		value: value,
	}

	mw.tail = mw.tail.next

	mw.length++
}

func (mw *MovingWindow) deq() float64 {
	if mw.tail == mw.root {
		return math.NaN()
	}

	value := mw.root.next.value
	mw.root.next = mw.root.next.next

	mw.length--

	return value
}

func (mw *MovingWindow) calcRaw(new, old float64) {
	mw.variance = mw.variance + (new * new) - (old * old)
	mw.sum = mw.sum + (new - old)
}

func (mw *MovingWindow) calcFinal() (float64, float64) {
	if mw.length > 0 {
		mw.average = mw.sum / float64(mw.length)

		if mw.length > 1 {
			// mw.stddev = math.Sqrt(mw.variance / (mw.length - 1))
			// okay so this came from dpayne/cli-visualizer
			mw.stddev = (mw.variance / float64(mw.length)) - math.Pow(mw.average, 2)
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
	mw.enq(value)

	if mw.length > mw.capacity {
		mw.calcRaw(value, mw.deq())
	} else {
		mw.calcRaw(value, 0)
	}

	return mw.calcFinal()
}

// Drop removes count items from the window
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	for ; count > 0 && mw.length > 0; count-- {
		mw.calcRaw(0, mw.deq())
	}

	// If we dont have enough length for standard dev, clear variance
	if mw.length < 2 {
		mw.variance = 0
		if mw.length < 1 {
			// mw.length = 0
			// same idea with sum. just clear it so we dont have a rouding issue
			mw.sum = 0
		}
	}

	return mw.calcFinal()
}

// Len returns how many items in the window
func (mw *MovingWindow) Len() int {
	return int(mw.length)
}

// Cap returns max size of window
func (mw *MovingWindow) Cap() int {
	return int(mw.capacity)
}

// IsEmpty checks for window emptiness
func (mw *MovingWindow) IsEmpty() bool {
	return mw.tail == mw.root
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
