package graphic

import "github.com/nsf/termbox-go"

func drawUp(bins [][]float64, count int, scale float64, state State, cfg Config) error {

	var vHeight = state.Height - cfg.BaseThick
	if vHeight < 0 {
		vHeight = 0
	}

	scale = float64(vHeight) / scale

	var cPaddedWidth = (cfg.BinWidth * count * len(bins)) - cfg.SpaceWidth

	if cPaddedWidth > state.Width || cPaddedWidth < 0 {
		cPaddedWidth = state.Width
	}

	var xCol = (state.Width - cPaddedWidth) / 2

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

			var xRow = state.Height

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
