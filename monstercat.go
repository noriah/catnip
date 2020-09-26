package tavis

import "math"

// Monstercat implements monstercat audio filter
type Monstercat struct {
	bars BarBuffer
}

func (mc *Monstercat) Waves(num int, waves int) {
	if waves <= 0 {
		return
	}
}

func (mc *Monstercat) Smooth(num int, factor float64) {
	if factor <= 1 {
		return
	}

	var (
		pass int
		bar  int
	)

	for pass = 0; pass < num; pass++ {
		for bar = pass - 1; bar >= 0; bar-- {
			mc.bars[bar] = max(mc.bars[pass]/pow(factor, pass-bar), mc.bars[bar])
		}

		for bar = pass + 1; bar < num; bar++ {
			mc.bars[bar] = max(mc.bars[pass]/pow(factor, bar-pass), mc.bars[bar])
		}
	}
}

func max(bar, baz BarType) BarType {
	if bar < baz {
		return baz
	}
	return bar
}

func pow(factor float64, delta int) BarType {
	return BarType(math.Pow(factor, float64(delta)))
}
