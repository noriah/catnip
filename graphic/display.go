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

	BarRuneV = '\u2580'
	BarRune  = '\u2588'
	BarRuneH = '\u2590'

	StyleReverse = termbox.AttrReverse

	// NumRunes number of runes for sub step bars
	NumRunes = 8
)

// DrawType is the type.
type DrawType int

// draw types
const (
	DrawMin DrawType = iota
	DrawUp
	DrawUpDown
	DrawDown
	DrawLeftRight
	DrawMax

	// DrawDefault is the default draw type.
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

// Display handles drawing our visualizer.
type Display struct {
	running     uint32
	barSize     int
	spaceSize   int
	binSize     int
	baseSize    int
	termWidth   int
	termHeight  int
	drawType    DrawType
	styles      Styles
	styleBuffer []termbox.Attribute
}

// Init initializes the display.
// Should be called before any other display method.
func (d *Display) Init() error {
	// make a large buffer as this could be as big as the screen width/height.
	d.styleBuffer = make([]termbox.Attribute, 4096)

	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputAlt)
	termbox.SetOutputMode(termbox.Output256)
	termbox.HideCursor()

	d.termWidth, d.termHeight = termbox.Size()

	return nil
}

// Close will stop display and clean up the terminal.
func (d *Display) Close() error {
	termbox.Close()
	return nil
}

// Start display is bad.
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
					d.AdjustSizes(1, 0)

				case termbox.KeyArrowRight:
					d.AdjustSizes(0, 1)

				case termbox.KeyArrowDown:
					d.AdjustSizes(-1, 0)

				case termbox.KeyArrowLeft:
					d.AdjustSizes(0, -1)

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
				d.updateStyleBuffer()

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

func intMax(x1, x2 int) int {
	if x1 < x2 {
		return x2
	}
	return x1
}

func intMin(x1, x2 int) int {
	if x1 > x2 {
		return x2
	}
	return x1
}

func (d *Display) updateStyleBuffer() {
	switch d.drawType {
	case DrawUp:
		d.fillStyleBuffer(d.termHeight-d.baseSize, d.baseSize, 0)

	case DrawUpDown:
		centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
		centerStop := centerStart + d.baseSize
		d.fillStyleBuffer(centerStart, d.baseSize, d.termHeight-centerStop)

	case DrawDown:
		d.fillStyleBuffer(0, d.baseSize, d.termHeight-d.baseSize)

	case DrawLeftRight:
		centerStart := intMax((d.termWidth-d.baseSize)/2, 0)
		centerStop := centerStart + d.baseSize
		d.fillStyleBuffer(centerStart, d.baseSize, d.termWidth-centerStop)
	}
}

func (d *Display) fillStyleBuffer(left, center, right int) {
	i := 0
	for stop := left; i < stop; i++ {
		d.styleBuffer[i] = d.styles.Foreground
	}

	for stop := i + center; i < stop; i++ {
		d.styleBuffer[i] = d.styles.CenterLine
	}

	for stop := i + right; i < stop; i++ {
		d.styleBuffer[i] = d.styles.Foreground
	}
}

// Stop display not work.
func (d *Display) Stop() error {
	if atomic.CompareAndSwapUint32(&d.running, 1, 0) {
		termbox.Interrupt()
	}

	return nil
}

// Draw takes data and draws.
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

// SetSizes takes a bar size and spacing size.
// Returns number of bars able to show.
func (d *Display) SetSizes(bar, space int) {
	bar = intMax(bar, 1)
	space = intMax(space, 0)

	d.barSize = bar
	d.spaceSize = space
	d.binSize = bar + space
}

// AdjustSizes modifies the bar and space size by barDelta and spaceDelta.
func (d *Display) AdjustSizes(barDelta, spaceDelta int) {
	d.SetSizes(d.barSize+barDelta, d.spaceSize+spaceDelta)
}

// SetBase will set the base size.
func (d *Display) SetBase(size int) {
	size = intMax(size, 0)
	d.baseSize = size

	d.updateStyleBuffer()
}

// AdjustBase will change the base by delta units
func (d *Display) AdjustBase(delta int) {
	d.SetBase(d.baseSize + delta)
}

func (d *Display) SetStyles(styles Styles) {
	d.styles = styles

	d.updateStyleBuffer()
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

	d.updateStyleBuffer()
}

// Bars returns the number of bars we will draw.
func (d *Display) Bars(sets ...int) int {
	var x = 1
	if len(sets) > 0 {
		x = sets[0]
	}

	switch d.drawType {
	case DrawUp, DrawDown:
		return (d.termWidth / d.binSize) / x
	case DrawUpDown:
		return d.termWidth / d.binSize
	case DrawLeftRight:
		return d.termHeight / d.binSize
	default:
		return 0
	}
}

func sizeAndCap(value float64, space int, zeroBase bool, baseRune rune) (int, rune) {
	var steps, stop = int(value * NumRunes), space * NumRunes

	if zeroBase {
		if steps < stop {
			return space - (steps / NumRunes), baseRune + rune(steps%NumRunes)
		}

		return 0, baseRune
	}

	if steps < stop {
		return steps / NumRunes, baseRune - rune(steps%NumRunes)
	}

	return space, baseRune
}

// DRAWING METHODS

// DrawUp will draw up.
func (d *Display) DrawUp(bins [][]float64, count int, scale float64) {

	barSpace := intMax(d.termHeight-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * count * len(bins)) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * count
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < count; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((count - 1) - xBar) * xSet)
			start, bCap := sizeAndCap(chBins[xBin]*scale, barSpace, true, BarRuneV)

			xCol := (xBar * d.binSize) + (channelWidth * xSet) + edgeOffset
			lCol := xCol + d.barSize

			for ; xCol < lCol; xCol++ {

				if bCap > BarRuneV {
					termbox.SetCell(xCol, start-1, bCap, d.styles.Foreground, d.styles.Background)
				}

				for xRow := start; xRow < d.termHeight; xRow++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xRow], d.styles.Background)
				}
			}
		}
	}
}

// DrawDown will draw down.
func (d *Display) DrawDown(bins [][]float64, count int, scale float64) {

	barSpace := intMax(d.termHeight-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * count * len(bins)) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * count
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < count; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((count - 1) - xBar) * xSet)
			stop, bCap := sizeAndCap(chBins[xBin]*scale, barSpace, false, BarRune)
			if stop += d.baseSize; stop >= d.termHeight {
				stop = d.termHeight
				bCap = BarRune
			}

			xCol := (xBar * d.binSize) + (channelWidth * xSet) + edgeOffset
			lCol := xCol + d.barSize

			for ; xCol < lCol; xCol++ {

				for xRow := 0; xRow < stop; xRow++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xRow], d.styles.Background)
				}

				if bCap < BarRune {
					termbox.SetCell(xCol, stop, bCap, StyleReverse, d.styles.Foreground)
				}
			}
		}
	}
}

// DrawUpDown will draw up and down.
func (d *Display) DrawUpDown(bins [][]float64, count int, scale float64) {

	centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termHeight-centerStop)) / scale

	edgeOffset := intMax((d.termWidth-((d.binSize*count)-d.spaceSize))/2, 0)

	setCount := len(bins)

	for xBar := 0; xBar < count; xBar++ {

		lStart, lCap := sizeAndCap(bins[0][xBar]*scale, centerStart, true, BarRuneV)
		rStop, rCap := sizeAndCap(bins[1%setCount][xBar]*scale, centerStart, false, BarRune)
		if rStop += centerStop; rStop >= d.termHeight {
			rStop = d.termHeight
			rCap = BarRune
		}

		xCol := xBar*d.binSize + edgeOffset
		lCol := intMin(xCol+d.barSize, d.termWidth)

		for ; xCol < lCol; xCol++ {

			if lCap > BarRuneV {
				termbox.SetCell(xCol, lStart-1, lCap, d.styles.Foreground, d.styles.Background)
			}

			for xRow := lStart; xRow < rStop; xRow++ {
				termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xRow], d.styles.Background)
			}

			// last part of right bars.
			if rCap < BarRune {
				termbox.SetCell(xCol, rStop, rCap, StyleReverse, d.styles.Foreground)
			}
		}
	}
}

// DrawLeftRight will draw left and right.
func (d *Display) DrawLeftRight(bins [][]float64, count int, scale float64) {
	centerStart := intMax((d.termWidth-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termWidth-centerStop)) / scale

	edgeOffset := intMax((d.termHeight-((d.binSize*count)-d.spaceSize))/2, 0)

	setCount := len(bins)

	for xBar := 0; xBar < count; xBar++ {

		// draw higher frequencies at the top
		xBin := count - 1 - xBar

		lStart, lCap := sizeAndCap(bins[0][xBin]*scale, centerStart, true, BarRune)
		rStop, rCap := sizeAndCap(bins[1%setCount][xBin]*scale, centerStart, false, BarRuneH)
		if rStop += centerStop; rStop >= d.termWidth {
			rStop = d.termWidth
			rCap = BarRuneH
		}

		xRow := xBar*d.binSize + edgeOffset
		lRow := intMin(xRow+d.barSize, d.termHeight)

		for ; xRow < lRow; xRow++ {

			if lCap > BarRune {
				termbox.SetCell(lStart-1, xRow, lCap, StyleReverse, d.styles.Background)
			}

			for xCol := lStart; xCol < rStop; xCol++ {
				termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xCol], d.styles.Background)
			}

			if rCap < BarRuneH {
				termbox.SetCell(rStop, xRow, rCap, d.styles.Foreground, d.styles.Foreground)
			}
		}
	}
}
