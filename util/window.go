package util

import "math"

type MovingWindow struct {
	vals chan float64
	sum  float64
	size float64
	sd   float64
}

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

func (mw *MovingWindow) Update(newVal float64) (float64, float64) {
	val := <-mw.vals
	mw.sd -= math.Pow(val-mw.Mean(), 2)
	mw.sum -= val
	mw.sum += newVal
	mw.sd += math.Pow(newVal-mw.Mean(), 2)
	mw.vals <- newVal
	return mw.Mean(), mw.StdDev()
}

func (mw *MovingWindow) Mean() float64 {
	return mw.sum / mw.size
}

func (mw *MovingWindow) StdDev() float64 {
	return math.Sqrt(mw.sd / mw.size)
}
