package graphic

import (
	"context"
	"sync/atomic"

	"github.com/nsf/termbox-go"
)

// Constants
const (

	// Bar Constants

	SpaceRune = '\u0020'

	BarRuneR = '\u2580'
	BarRune  = '\u2588'
	BarRuneS = '\u2590'

	StyleReverse = termbox.AttrReverse

	// NumRunes number of runes for sub step bars
	NumRunes = 8
)

// DrawType is the type
type DrawType int

// draw types
const (
	DrawMin DrawType = iota
	DrawDown
	DrawUpDown
	DrawUp
	DrawLeftRight
	DrawMax

	// DrawDefault is the default draw type
	DrawDefault = DrawUpDown
)

// Styles is the structure for the styles that Display will draw using.
type Styles struct {
	Foreground termbox.Attribute
	Background termbox.Attribute
	CenterLine termbox.Attribute
}

// DefaultStyles returns the default styles.
func DefaultStyles() Styles {
	return Styles{
		Foreground: termbox.ColorDefault,
		Background: termbox.ColorDefault,
		CenterLine: termbox.ColorMagenta,
	}
}

// StylesFromUInt16 converts 3 uint16 values to styles.
func StylesFromUInt16(fg, bg, center uint16) Styles {
	return Styles{
		Foreground: termbox.Attribute(fg),
		Background: termbox.Attribute(bg),
		CenterLine: termbox.Attribute(center),
	}
}

// AsUInt16s converts the styles to 3 uint16 values.
func (s Styles) AsUInt16s() (fg, bg, center uint16) {
	fg = uint16(s.Foreground)
	bg = uint16(s.Background)
	center = uint16(s.CenterLine)
	return
}

// Display handles drawing our visualizer
type Display struct {
	running    uint32
	barWidth   int
	spaceWidth int
	binWidth   int
	baseThick  int
	termWidth  int
	termHeight int
	drawType   DrawType
	styles     Styles
}

// Init initializes the display
func (d *Display) Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputAlt)
	termbox.SetOutputMode(termbox.Output256)
	termbox.HideCursor()

	d.termWidth, d.termHeight = termbox.Size()

	return nil
}

// Close will stop display and clean up the terminal
func (d *Display) Close() error {
	termbox.Close()
	return nil
}

// Start display is bad
func (d *Display) Start(ctx context.Context) context.Context {
	var dispCtx, dispCancel = context.WithCancel(ctx)

	go func(ctx context.Context, fn context.CancelFunc, d *Display) {

		defer fn()

		atomic.StoreUint32(&d.running, 1)
		defer atomic.StoreUint32(&d.running, 0)

		for {

			var ev = termbox.PollEvent()

			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {

				case termbox.KeyArrowUp:
					d.AdjustWidths(1, 0)

				case termbox.KeyArrowRight:
					d.AdjustWidths(0, 1)

				case termbox.KeyArrowDown:
					d.AdjustWidths(-1, 0)

				case termbox.KeyArrowLeft:
					d.AdjustWidths(0, -1)

				case termbox.KeySpace:
					d.SetDrawType(d.drawType + 1)

				case termbox.KeyCtrlC:
					return
				default:

					switch ev.Ch {
					case '+', '=':
						d.AdjustBase(1)

					case '-', '_':
						d.AdjustBase(-1)

					case 'q', 'Q':
						return

					default:

					} // switch ev.Ch

				} // switch ev.Key

			case termbox.EventResize:
				d.termWidth = ev.Width
				d.termHeight = ev.Height

			case termbox.EventInterrupt:
				return

			default:

			} // switch ev.Type

			// check if we need to exit
			select {
			case <-ctx.Done():
				return
			default:
			}

		} // for

	}(dispCtx, dispCancel, d)

	return dispCtx
}

// Stop display not work
func (d *Display) Stop() error {
	if atomic.CompareAndSwapUint32(&d.running, 1, 0) {
		termbox.Interrupt()
	}

	return nil
}

// Draw takes data and draws
func (d *Display) Draw(bufs [][]float64, channels, count int, scale float64) error {

	switch d.drawType {
	case DrawUp:
		d.DrawUp(bufs, count, scale)
	case DrawUpDown:
		d.DrawUpDown(bufs, count, scale)
	case DrawDown:
		d.DrawDown(bufs, count, scale)
	case DrawLeftRight:
		d.DrawLeftRight(bufs, count, scale)
	default:
		return nil
	}

	termbox.Flush()

	termbox.Clear(d.styles.Foreground, d.styles.Background)

	return nil
}

// SetWidths takes a bar width and spacing width
// Returns number of bars able to show
func (d *Display) SetWidths(bar, space int) {

	if bar < 1 {
		bar = 1
	}

	if space < 0 {
		space = 0
	}

	d.barWidth = bar
	d.spaceWidth = space
	d.binWidth = bar + space
}

// AdjustWidths modifies the bar and space width by barDelta and spaceDelta
func (d *Display) AdjustWidths(barDelta, spaceDelta int) {
	d.SetWidths(d.barWidth+barDelta, d.spaceWidth+spaceDelta)
}

// SetBase will set the base thickness
func (d *Display) SetBase(thick int) {
	switch {

	case thick < 0:
		d.baseThick = 0

	default:
		d.baseThick = thick

	}
}

func (d *Display) SetStyles(styles Styles) {
	// if styles.Background > 266 {
	// }
	d.styles = styles
}

// AdjustBase will change the base by delta units
func (d *Display) AdjustBase(delta int) {
	d.SetBase(d.baseThick + delta)
}

// SetDrawType sets the draw type for future draws
func (d *Display) SetDrawType(dt DrawType) {
	switch {
	case dt <= DrawMin:
		d.drawType = DrawMax - 1
	case dt >= DrawMax:
		d.drawType = DrawMin + 1
	default:
		d.drawType = dt
	}
}

// Bars returns the number of bars we will draw
func (d *Display) Bars(sets ...int) int {
	var x = 1
	if len(sets) > 0 {
		x = sets[0]
	}

	switch d.drawType {
	case DrawUp, DrawDown:
		return (d.termWidth / d.binWidth) / x
	case DrawUpDown:
		return d.termWidth / d.binWidth
	case DrawLeftRight:
		return d.termHeight / d.binWidth
	default:
		return 0
	}
}

// Dims returns screen dimensions
func (d *Display) Dims(sets ...int) (int, int) {
	return d.Bars(sets...), d.termHeight
}

func stopAndTop(value float64, height int, up bool) (int, rune) {
	var stop, h = int(value * NumRunes), height * NumRunes

	if up {

		if stop < h {
			return height - (stop / NumRunes), BarRuneR + rune(stop%NumRunes)
		}

		return 0, BarRuneR
	}

	if stop < h {
		return stop / NumRunes, BarRune - rune(stop%NumRunes)
	}

	return height, BarRune
}

// DRAWING METHODS

// DrawUp will draw up
func (d *Display) DrawUp(bins [][]float64, count int, scale float64) error {

	var vHeight = d.termHeight - d.baseThick
	if vHeight < 0 {
		vHeight = 0
	}

	scale = float64(vHeight) / scale

	var cPaddedWidth = (d.binWidth * count * len(bins)) - d.spaceWidth

	if cPaddedWidth > d.termWidth || cPaddedWidth < 0 {
		cPaddedWidth = d.termWidth
	}

	var xCol = (d.termWidth - cPaddedWidth) / 2

	var delta = 1
	var xBin int
	// var xBin = count - 1

	for _, chBins := range bins {
		var stop, top = stopAndTop(chBins[xBin]*scale, vHeight, true)

		var lCol = xCol + d.barWidth
		var lColMax = xCol + (d.binWidth * count) - d.spaceWidth

		for ; ; xCol++ {
			if xCol >= lCol {
				if xCol >= lColMax {
					break
				}

				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = stopAndTop(chBins[xBin]*scale, vHeight, true)

				xCol += d.spaceWidth
				lCol = xCol + d.barWidth
			}

			var xRow = d.termHeight

			for ; xRow >= vHeight; xRow-- {
				termbox.SetCell(xCol, xRow, BarRune, d.styles.CenterLine, d.styles.Background)
			}

			for ; xRow >= stop; xRow-- {
				termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
			}

			if top > BarRuneR {
				termbox.SetCell(xCol, xRow, top, d.styles.Foreground, d.styles.Background)
			}
		}

		xCol += d.spaceWidth
		delta = -delta
	}

	return nil
}

// DrawDown will draw down
func (d *Display) DrawDown(bins [][]float64, count int, scale float64) error {

	var vHeight = d.termHeight - d.baseThick
	if vHeight < 0 {
		vHeight = 0
	}

	scale = float64(vHeight) / scale

	var cPaddedWidth = (d.binWidth * count * len(bins)) - d.spaceWidth

	if cPaddedWidth > d.termWidth || cPaddedWidth < 0 {
		cPaddedWidth = d.termWidth
	}

	var xBin int
	var xCol = (d.termWidth - cPaddedWidth) / 2
	var delta = 1

	for _, chBins := range bins {
		var stop, top = stopAndTop(chBins[xBin]*scale, vHeight, false)
		if stop += d.baseThick; stop >= d.termHeight {
			stop = d.termHeight
			top = BarRune
		}

		var lCol = xCol + d.barWidth
		var lColMax = xCol + (d.binWidth * count) - d.spaceWidth

		for {
			if xCol >= lCol {
				if xCol >= lColMax {
					break
				}

				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = stopAndTop(chBins[xBin]*scale, vHeight, false)
				if stop += d.baseThick; stop >= d.termHeight {
					stop = d.termHeight
					top = BarRune
				}

				xCol += d.spaceWidth
				lCol = xCol + d.barWidth
			}

			var xRow = 0

			for ; xRow < d.baseThick; xRow++ {
				termbox.SetCell(xCol, xRow, BarRune, d.styles.CenterLine, d.styles.Background)
			}

			for ; xRow < stop; xRow++ {
				termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
			}

			if top < BarRune {
				termbox.SetCell(xCol, xRow, top, StyleReverse, d.styles.Foreground)
			}

			xCol++
		}

		xCol += d.spaceWidth
		delta = -delta
	}

	return nil
}

// DrawUpDown will draw up and down
func (d *Display) DrawUpDown(bins [][]float64, count int, scale float64) error {
	var cSetCount = len(bins)

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window, always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)

	var centerStart = (d.termHeight - d.baseThick) / 2
	if centerStart < 0 {
		centerStart = 0
	}

	var centerStop = centerStart + d.baseThick

	scale = float64(centerStart) / scale

	var xBin = 0

	var xCol = (d.termWidth - ((d.binWidth * count) - d.spaceWidth)) / 2

	if xCol < 0 {
		xCol = 0
	}

	// TODO(nora): benchmark

	var lStop, lTop = stopAndTop(bins[0][xBin]*scale, centerStart, true)
	var rStop, rTop = stopAndTop(bins[1%cSetCount][xBin]*scale, centerStart, false)
	if rStop += centerStop; rStop >= d.termHeight {
		rStop = d.termHeight
		rTop = BarRune
	}

	var lCol = xCol + d.barWidth

	for ; ; xCol++ {

		if xCol >= lCol {

			if xCol >= d.termWidth {
				break
			}

			if xBin++; xBin >= count {
				break
			}

			lStop, lTop = stopAndTop(bins[0][xBin]*scale, centerStart, true)
			rStop, rTop = stopAndTop(bins[1%cSetCount][xBin]*scale, centerStart, false)
			if rStop += centerStop; rStop >= d.termHeight {
				rStop = d.termHeight
				rTop = BarRune
			}

			xCol += d.spaceWidth
			lCol = xCol + d.barWidth
		}

		var xRow = lStop

		if lTop > BarRuneR {
			termbox.SetCell(xCol, xRow-1, lTop, d.styles.Foreground, d.styles.Background)
		}

		for ; xRow < centerStart; xRow++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
		}

		// center line
		for ; xRow < centerStop; xRow++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.CenterLine, d.styles.Background)
		}

		// right bars go down
		for ; xRow < rStop; xRow++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
		}

		// last part of right bars.
		if rTop < BarRune {
			termbox.SetCell(xCol, xRow, rTop, StyleReverse, d.styles.Foreground)
		}
	}

	return nil
}

func sizeAndCap(value float64, width int, right bool) (int, rune) {
	var size, w = int(value * NumRunes), width * NumRunes

	if right {

		if size < w {
			return size / NumRunes, BarRuneS - rune(size%NumRunes)
		}

		return width, BarRuneS
	}

	if size < w {
		return width - (size / NumRunes), BarRune + rune(size%NumRunes)
	}

	return 0, BarRune
}

// DrawLeftRight will draw left and right
func (d *Display) DrawLeftRight(bins [][]float64, count int, scale float64) error {
	var cSetCount = len(bins)

	var centerStart = (d.termWidth - d.baseThick) / 2
	if centerStart < 0 {
		centerStart = 0
	}

	var centerStop = centerStart + d.baseThick

	scale = float64(centerStart) / scale

	var xBin = count - 1

	var xRow = (d.termHeight - ((d.binWidth * count) - d.spaceWidth)) / 2

	if xRow < 0 {
		xRow = 0
	}

	// TODO(nora): benchmark

	var lStart, lCap = sizeAndCap(bins[0][xBin]*scale, centerStart, false)
	var rStop, rCap = sizeAndCap(bins[1%cSetCount][xBin]*scale, centerStart, true)
	if rStop += centerStop; rStop >= d.termWidth {
		rStop = d.termWidth
		rCap = BarRune
	}

	var lRow = xRow + d.barWidth

	for ; ; xRow++ {

		if xRow >= lRow {

			if xRow >= d.termHeight {
				break
			}

			if xBin--; xBin < 0 {
				break
			}

			lStart, lCap = sizeAndCap(bins[0][xBin]*scale, centerStart, false)
			rStop, rCap = sizeAndCap(bins[1%cSetCount][xBin]*scale, centerStart, true)
			if rStop += centerStop; rStop >= d.termWidth {
				rStop = d.termWidth
				rCap = BarRune
			}

			xRow += d.spaceWidth
			lRow = xRow + d.barWidth
		}

		var xCol = lStart

		if lCap > BarRune {
			termbox.SetCell(xCol-1, xRow, lCap, StyleReverse, d.styles.Background)
		}

		for ; xCol < centerStart; xCol++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
		}

		// center line
		for ; xCol < centerStop; xCol++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.CenterLine, d.styles.Background)
		}

		// right bars go down
		for ; xCol < rStop; xCol++ {
			termbox.SetCell(xCol, xRow, BarRune, d.styles.Foreground, d.styles.Background)
		}

		// last part of right bars.
		if rCap < BarRuneS {
			termbox.SetCell(xCol, xRow, rCap, d.styles.Foreground, d.styles.Foreground)
		}
	}

	return nil
}
