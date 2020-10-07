package n2s3

import "math"

type attr struct {
	value float64
	delta float64
}

func (a *attr) setV(v float64) {

}

func (a *attr) set(value interface{}) float64 {
	a.delta = attrGetFloat(value) - a.value
	a.value += a.delta
	return a.delta
}

func (a *attr) add(value interface{}) float64 {
	a.delta = attrGetFloat(value)
	a.value += a.delta

	return a.value
}

func (a *attr) addZero(value interface{}) float64 {
	if a.add(value) < 0 {
		a.delta -= a.value
		a.value = 0
	}

	return a.value
}

func (a *attr) sub(value interface{}) float64 {
	a.delta = -attrGetFloat(value)
	a.value += a.delta

	return a.value
}

func (a *attr) mul(value interface{}) float64 {
	a.delta = a.value
	a.value *= attrGetFloat(value)
	a.delta = a.value - a.delta

	return a.value
}

func (a *attr) div(value interface{}) float64 {
	a.delta = a.value
	a.value /= attrGetFloat(value)
	a.delta = a.value - a.delta

	return a.value
}

func attrGetFloat(value interface{}) float64 {
	switch v := value.(type) {
	case *attr:
		return v.value
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return math.NaN()
	}
}
