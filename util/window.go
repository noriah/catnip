package util

import (
	"math"
)

// as long as we know what comes next, we can keep a chain
type node struct {
	value float64
	next  *node
}

// MovingWindow is a moving window
//
// we only keep a reference to the tail node
// the tail node is not a value hold node, but is referenced by the last
// valid node, and points to the "first" valid node.
// we add new items to the window by setting the tail "value" attribute,
// make a new node, point that node "next" to the first valid value.
// then set the current tail to point to this new node
// then set our tail to be this new node
//
// removal of a value is much simpler, remove the node from the list
// by removing the "first" node. set the current tail to point to the
// node referenced by this removed node
// return value of removed node
type MovingWindow struct {
	tail *node

	pool []*node

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
		tail:     &node{},
		pool:     make([]*node, size),
		capacity: size,
		sum:      0.0,
		average:  0.0,
	}

	for xNode := 0; xNode < size; xNode++ {
		mw.pool[xNode] = &node{}
	}

	mw.tail.next = mw.tail

	return mw
}

func (mw *MovingWindow) calcFinal() (float64, float64) {
	if mw.length > 1 {
		// mw.stddev = math.Sqrt(mw.variance / (mw.length - 1))
		// okay so this came from dpayne/cli-visualizer
		mw.stddev = (mw.variance / float64(mw.length-1)) - (mw.average * mw.average)
		mw.stddev = math.Sqrt(math.Abs(mw.stddev))
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
	if mw.length < mw.capacity {

		mw.pool[mw.length].next = mw.tail.next
		mw.tail.next = mw.pool[mw.length]

		mw.length++

		mw.variance += value * value
		mw.sum += value

	} else {
		mw.variance += (value * value) - (mw.tail.next.value * mw.tail.next.value)
		mw.sum += value - mw.tail.next.value
	}

	mw.tail.value = value
	mw.tail = mw.tail.next

	return mw.calcFinal()
}

// Drop removes count items from the window
func (mw *MovingWindow) Drop(count int) (float64, float64) {
	if mw.length <= 0 {
		return mw.calcFinal()
	}

	for count > 0 && mw.length > 0 {
		mw.sum -= mw.tail.next.value
		mw.variance -= mw.tail.next.value * mw.tail.next.value

		mw.length--

		mw.pool[mw.length] = mw.tail.next
		mw.tail.next = mw.tail.next.next

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
func (mw *MovingWindow) Stats() (float64, float64) {
	return mw.average, mw.stddev
}
