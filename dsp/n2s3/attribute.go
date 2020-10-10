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

func (a *attr) addClamp(min, max float64, value interface{}) float64 {
	if x := math.Min(max, math.Max(min, a.add(value))); x != a.value {
		a.delta += x - a.value
		a.value = x
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
	case attr:
		return v.value
	case *attr:
		return v.value
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}
