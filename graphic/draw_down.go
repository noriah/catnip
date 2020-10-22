package graphic

import "github.com/nsf/termbox-go"

func drawDown(bins [][]float64, count int, cfg Config, scale float64) error {
	var cSetCount = len(bins)

	var cWidth, cHeight = termbox.Size()

	var cChanWidth = (cfg.BinWidth * count) - cfg.SpaceWidth
	var cPaddedWidth = (cfg.BinWidth * count * cSetCount) - cfg.SpaceWidth
	var cOffset = (cWidth - cPaddedWidth) / 2

	var vHeight = cHeight - cfg.BaseThick

	scale = float64(vHeight) / scale

	var xBin int
	var xCol = cOffset
	var delta = 1

	for xCh := range bins {
		var stop, top = stopAndTop(bins[xCh][xBin]*scale, vHeight, false)

		var lCol = xCol + cfg.BarWidth
		var lColMax = xCol + cChanWidth

		for xCol < lColMax {

			if xCol >= lCol {
				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = stopAndTop(bins[xCh][xBin]*scale, vHeight, false)

				xCol += cfg.SpaceWidth
				lCol = xCol + cfg.BarWidth
			}

			var xRow = 0

			for xRow < cfg.BaseThick {
				termbox.SetCell(xCol, xRow, BarRune, StyleCenter, StyleDefaultBack)

				xRow++
			}

			for xRow < stop {
				termbox.SetCell(xCol, xRow, BarRune, StyleDefault, StyleDefaultBack)

				xRow++
			}

			if top < BarRune {
				termbox.SetCell(xCol, xRow, top, StyleReverse, StyleDefault)
			}

			xCol++
		}

		xCol += cfg.SpaceWidth
		delta = -delta
	}

	return nil
}

// var stopAndTop = func(value float64) (stop int, top int) {
// 	if value *= scale; value < float64(vHeight) {
// 		top = int(value * NumRunes)
// 	} else {
// 		top = vHeight * NumRunes
// 	}
// 	stop = (top / NumRunes) + cfg.BaseThick
// 	top %= NumRunes

// 	if stop > cHeight {
// 		stop = cHeight
// 		top = 0
// 	}

// 	return
// }
