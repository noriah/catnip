package graphic

import (
	"context"
	"sync/atomic"

	"github.com/noriah/catnip/util"

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

	// ScalingWindow in seconds
	ScalingWindow = 1.5
	// PeakThreshold is the threshold to not draw if the peak is less.
	PeakThreshold = 0.001
)

// DrawType is the type.
type DrawType int

// draw types
const (
	DrawMin DrawType = iota
	DrawUp
	DrawUpDown
	DrawDown
	DrawLeft
	DrawLeftRight
	DrawRight
	DrawUpDownSplit
	DrawLeftRightSplit
	DrawUpDownSplitVert
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
	trackZero   int
	invertDraw  bool
	window      *util.MovingWindow
	drawType    DrawType
	styles      Styles
	styleBuffer []termbox.Attribute
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewDisplay() *Display {
	return &Display{}
}

// Init initializes the display.
// Should be called before any other display method.
func (d *Display) Init(sampleRate float64, sampleSize int) error {
	// make a large buffer as this could be as big as the screen width/height.

	windowSize := ((int(ScalingWindow * sampleRate)) / sampleSize) * 2
	d.window = util.NewMovingWindow(windowSize)

	d.styleBuffer = make([]termbox.Attribute, 4096)

	// Prevent crash on Tmux.
	prevState, err := normalizeTerminal()
	if err != nil {
		return err
	}
	defer prevState()

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
	d.ctx, d.cancel = context.WithCancel(ctx)

	go d.inputProcessor()

	return d.ctx
}

// Stop display not work.
func (d *Display) Stop() error {
	if atomic.CompareAndSwapUint32(&d.running, 1, 0) {
		termbox.Interrupt()
	}

	return nil
}

// Draw takes data and draws.
func (d *Display) Write(buffers [][]float64, channels int) error {

	peak := 0.0
	bins := d.Bins(channels)

	for i := 0; i < channels; i++ {
		for _, val := range buffers[i][:bins] {
			if val > peak {
				peak = val
			}
		}
	}

	scale := 1.0

	if peak >= PeakThreshold {
		d.trackZero = 0

		// do some scaling if we are above the PeakThreshold
		d.window.Update(peak)

	} else {
		if d.trackZero++; d.trackZero == 5 {
			d.window.Recalculate()
		}
	}

	vMean, vSD := d.window.Stats()

	if t := vMean + (2.0 * vSD); t > 1.0 {
		scale = t
	}

	switch d.drawType {
	case DrawUp:
		d.drawUp(buffers, channels, scale)

	case DrawUpDown:
		d.drawUpDown(buffers, channels, scale)

	case DrawUpDownSplit:
		d.drawUpDownSplit(buffers, channels, scale)

	case DrawUpDownSplitVert:
		d.drawUpDownSplitVert(buffers, channels, scale)

	case DrawDown:
		d.drawDown(buffers, channels, scale)

	case DrawLeft:
		d.drawLeft(buffers, channels, scale)

	case DrawLeftRight:
		d.drawLeftRight(buffers, channels, scale)

	case DrawLeftRightSplit:
		d.drawLeftRightSplit(buffers, channels, scale)

	case DrawRight:
		d.drawRight(buffers, channels, scale)

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

func (d *Display) SetInvertDraw(invert bool) {
	d.invertDraw = invert
}

// Bins returns the number of bars we will draw.
func (d *Display) Bins(chCount int) int {

	switch d.drawType {
	case DrawUp, DrawDown:
		return (d.termWidth / d.binSize) / chCount
	case DrawUpDownSplit, DrawUpDownSplitVert:
		return (d.termWidth / d.binSize) / 2
	case DrawUpDown:
		return d.termWidth / d.binSize
	case DrawLeft, DrawRight:
		return (d.termHeight / d.binSize) / chCount
	case DrawLeftRightSplit:
		return (d.termHeight / d.binSize) / 2
	case DrawLeftRight:
		return d.termHeight / d.binSize
	default:
		return 0
	}
}

func (d *Display) inputProcessor() {
	if d.cancel != nil {
		defer d.cancel()
	}

	atomic.StoreUint32(&d.running, 1)
	defer atomic.StoreUint32(&d.running, 0)

	for {
		ev := termbox.PollEvent()

		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {

			case termbox.KeySpace:
				d.SetDrawType(d.drawType + 1)

			case termbox.KeyCtrlC:
				return

			default:
				switch ev.Ch {
				case 'w', 'W':
					d.AdjustSizes(1, 0)

				case 'd', 'D':
					d.AdjustSizes(0, 1)

				case 's', 'S':
					d.AdjustSizes(-1, 0)

				case 'a', 'A':
					d.AdjustSizes(0, -1)

				case 'i', 'I':
					d.SetInvertDraw(!d.invertDraw)

				case 'r', 'R':
					d.window.Drop(d.window.Cap())

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
		case <-d.ctx.Done():
			return
		default:
		}
	} // for
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

	case DrawUpDown, DrawUpDownSplit, DrawUpDownSplitVert:
		centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
		centerStop := centerStart + d.baseSize
		d.fillStyleBuffer(centerStart, d.baseSize, d.termHeight-centerStop)

	case DrawDown:
		d.fillStyleBuffer(0, d.baseSize, d.termHeight-d.baseSize)

	case DrawLeft:
		d.fillStyleBuffer(d.termWidth-d.baseSize, d.baseSize, 0)

	case DrawLeftRight, DrawLeftRightSplit:
		centerStart := intMax((d.termWidth-d.baseSize)/2, 0)
		centerStop := centerStart + d.baseSize
		d.fillStyleBuffer(centerStart, d.baseSize, d.termWidth-centerStop)

	case DrawRight:
		d.fillStyleBuffer(0, d.baseSize, d.termWidth-d.baseSize)
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

func sizeAndCap(value float64, space int, zeroBase bool, baseRune rune) (int, rune) {
	steps, stop := int(value*NumRunes), space*NumRunes

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

// drawUp will draw up.
func (d *Display) drawUp(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	barSpace := intMax(d.termHeight-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

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

// drawUpDown will draw up and down.
func (d *Display) drawUpDown(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termHeight-centerStop)) / scale

	edgeOffset := intMax((d.termWidth-((d.binSize*binCount)-d.spaceSize))/2, 0)

	setCount := channelCount

	for xBar := 0; xBar < binCount; xBar++ {

		lStart, lCap := sizeAndCap(bins[0][xBar]*scale, centerStart, true, BarRuneV)
		rStop, rCap := sizeAndCap(bins[1%setCount][xBar]*scale, centerStart, false, BarRune)
		if rStop += centerStop; rStop >= d.termHeight {
			rStop = d.termHeight
			rCap = BarRune
		}

		xCol := xBar
		if d.invertDraw {
			xCol = binCount - 1 - xCol
		}

		xCol = xCol*d.binSize + edgeOffset
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

// drawUpDownSplit will draw up and down split down the middle for left and
// right channels.
func (d *Display) drawUpDownSplit(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termHeight-centerStop)) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSide := 0; xSide < 2; xSide++ {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSide)) + (((binCount - 1) - xBar) * xSide)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

			start, tCap := sizeAndCap(bins[xSide%channelCount][xBin]*scale, centerStart, true, BarRuneV)
			stop, bCap := sizeAndCap(bins[xSide%channelCount][xBin]*scale, centerStart, false, BarRune)
			if stop += centerStop; stop >= d.termHeight {
				stop = d.termHeight
				bCap = BarRune
			}

			xCol := (xBar * d.binSize) + (channelWidth * xSide) + edgeOffset
			lCol := xCol + d.barSize

			for ; xCol < lCol; xCol++ {

				if tCap > BarRuneV {
					termbox.SetCell(xCol, start-1, tCap, d.styles.Foreground, d.styles.Background)
				}

				for xRow := start; xRow < stop; xRow++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xRow], d.styles.Background)
				}

				if bCap < BarRune {
					termbox.SetCell(xCol, stop, bCap, StyleReverse, d.styles.Foreground)
				}
			}
		}
	}
}

// drawUpDownSplitVert will draw up and down split down the middle for left and
// right channels.
func (d *Display) drawUpDownSplitVert(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	centerStart := intMax((d.termHeight-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termHeight-centerStop)) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSide := 0; xSide < 2; xSide++ {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSide)) + (((binCount - 1) - xBar) * xSide)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

			start, tCap := sizeAndCap(bins[0][xBin]*scale, centerStart, true, BarRuneV)
			stop, bCap := sizeAndCap(bins[1%channelCount][xBin]*scale, centerStart, false, BarRune)
			if stop += centerStop; stop >= d.termHeight {
				stop = d.termHeight
				bCap = BarRune
			}

			xCol := (xBar * d.binSize) + (channelWidth * xSide) + edgeOffset
			lCol := xCol + d.barSize

			for ; xCol < lCol; xCol++ {

				if tCap > BarRuneV {
					termbox.SetCell(xCol, start-1, tCap, d.styles.Foreground, d.styles.Background)
				}

				for xRow := start; xRow < stop; xRow++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xRow], d.styles.Background)
				}

				if bCap < BarRune {
					termbox.SetCell(xCol, stop, bCap, StyleReverse, d.styles.Foreground)
				}
			}
		}
	}
}

// drawDown will draw down.
func (d *Display) drawDown(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	barSpace := intMax(d.termHeight-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termWidth), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termWidth - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

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

func (d *Display) drawLeft(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	barSpace := intMax(d.termWidth-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termHeight), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termHeight - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

			start, bCap := sizeAndCap(chBins[xBin]*scale, barSpace, true, BarRune)

			xRow := (xBar * d.binSize) + (channelWidth * xSet) + edgeOffset
			lRow := xRow + d.barSize

			for ; xRow < lRow; xRow++ {

				if bCap > BarRune {
					termbox.SetCell(start-1, xRow, bCap, StyleReverse, d.styles.Background)
				}

				for xCol := start; xCol < d.termWidth; xCol++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xCol], d.styles.Background)
				}
			}
		}
	}
}

// drawLeftRight will draw left and right.
func (d *Display) drawLeftRight(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	centerStart := intMax((d.termWidth-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termWidth-centerStop)) / scale

	edgeOffset := intMax((d.termHeight-((d.binSize*binCount)-d.spaceSize))/2, 0)

	setCount := channelCount

	for xBar := 0; xBar < binCount; xBar++ {

		// draw higher frequencies at the top
		xBin := binCount - 1 - xBar

		lStart, lCap := sizeAndCap(bins[0][xBin]*scale, centerStart, true, BarRune)
		rStop, rCap := sizeAndCap(bins[1%setCount][xBin]*scale, centerStart, false, BarRuneH)
		if rStop += centerStop; rStop >= d.termWidth {
			rStop = d.termWidth
			rCap = BarRuneH
		}

		xRow := xBar
		if d.invertDraw {
			xRow = binCount - 1 - xRow
		}

		xRow = xRow*d.binSize + edgeOffset

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

// drawLeftRight will draw left and right.
func (d *Display) drawLeftRightSplit(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	centerStart := intMax((d.termWidth-d.baseSize)/2, 0)
	centerStop := centerStart + d.baseSize

	scale = float64(intMin(centerStart, d.termWidth-centerStop)) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termHeight), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termHeight - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

			start, lCap := sizeAndCap(chBins[xBin]*scale, centerStart, true, BarRune)
			stop, rCap := sizeAndCap(chBins[xBin]*scale, centerStart, false, BarRuneH)
			if stop += centerStop; stop >= d.termWidth {
				stop = d.termWidth
				rCap = BarRuneH
			}

			xRow := (xBar * d.binSize) + (channelWidth * xSet) + edgeOffset
			lRow := xRow + d.barSize

			for ; xRow < lRow; xRow++ {

				if lCap > BarRune {
					termbox.SetCell(start-1, xRow, lCap, StyleReverse, d.styles.Background)
				}

				for xCol := start; xCol < stop; xCol++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xCol], d.styles.Background)
				}

				if rCap < BarRuneH {
					termbox.SetCell(stop, xRow, rCap, d.styles.Foreground, d.styles.Foreground)
				}
			}
		}
	}
}

func (d *Display) drawRight(bins [][]float64, channelCount int, scale float64) {
	binCount := d.Bins(channelCount)
	barSpace := intMax(d.termWidth-d.baseSize, 0)
	scale = float64(barSpace) / scale

	paddedWidth := (d.binSize * binCount * channelCount) - d.spaceSize
	paddedWidth = intMax(intMin(paddedWidth, d.termHeight), 0)

	channelWidth := d.binSize * binCount
	edgeOffset := (d.termHeight - paddedWidth) / 2

	for xSet, chBins := range bins {

		for xBar := 0; xBar < binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = binCount - 1 - xBin
			}

			stop, bCap := sizeAndCap(chBins[xBin]*scale, barSpace, false, BarRuneH)
			if stop += d.baseSize; stop >= d.termWidth {
				stop = d.termWidth
				bCap = BarRune
			}

			xRow := (xBar * d.binSize) + (channelWidth * xSet) + edgeOffset
			lRow := xRow + d.barSize

			for ; xRow < lRow; xRow++ {

				for xCol := 0; xCol < stop; xCol++ {
					termbox.SetCell(xCol, xRow, BarRune, d.styleBuffer[xCol], d.styles.Background)
				}

				if bCap < BarRuneH {
					termbox.SetCell(stop, xRow, bCap, d.styles.Foreground, d.styles.Foreground)
				}
			}
		}
	}
}
