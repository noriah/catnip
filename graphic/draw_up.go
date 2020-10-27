package graphic

import "github.com/nsf/termbox-go"

func drawUp(bins [][]float64, count int, cfg Config, scale float64) error {
	var cWidth, cHeight = termbox.Size()

	var vHeight = cHeight - cfg.BaseThick
	if vHeight < 0 {
		vHeight = 0
	}

	scale = float64(vHeight) / scale

	var cPaddedWidth = (cfg.BinWidth * count * len(bins)) - cfg.SpaceWidth

	if cPaddedWidth > cWidth || cPaddedWidth < 0 {
		cPaddedWidth = cWidth
	}

	var xCol = (cWidth - cPaddedWidth) / 2

	var delta = 1
	var xBin int
	// var xBin = count - 1

	for xCh := range bins {
		var stop, top = stopAndTop(bins[xCh][xBin]*scale, vHeight, true)

		var lCol = xCol + cfg.BarWidth
		var lColMax = xCol + (cfg.BinWidth * count) - cfg.SpaceWidth

		for {
			if xCol >= lCol {
				if xCol >= lColMax {
					break
				}

				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = stopAndTop(bins[xCh][xBin]*scale, vHeight, true)

				xCol += cfg.SpaceWidth
				lCol = xCol + cfg.BarWidth
			}

			var xRow = cHeight

			for xRow >= vHeight {
				termbox.SetCell(xCol, xRow, BarRune, StyleCenter, StyleDefaultBack)

				xRow--
			}

			for xRow >= stop {
				termbox.SetCell(xCol, xRow, BarRune, StyleDefault, StyleDefaultBack)

				xRow--
			}

			if top > BarRuneR {
				termbox.SetCell(xCol, xRow, top, StyleDefault, StyleDefaultBack)
			}

			xCol++
		}

		xCol += cfg.SpaceWidth
		delta = -delta
	}

	return nil
}
