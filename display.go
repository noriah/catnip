package tavis

import (
	"errors"
	"math"
	"sync"

	"github.com/gdamore/tcell/v2"
)

const (
	// DisplayBar is the block we use for bars
	DisplayBar rune = '\u2588'

	// DisplaySpace is the block we use for space (if we were to print one)
	// DisplaySpace rune = '\u0020'

	// MaxWidth will be removed at some point
	MaxWidth = 5000
)

// bar blocks for later
var (
	barHeightRunes = [...]rune{
		'\u0020',
		'\u2581',
		'\u2582',
		'\u2583',
		'\u2584',
		'\u2585',
		'\u2586',
		'\u2587',
		DisplayBar,
	}

	numRunes = len(barHeightRunes)

	styleDefault = tcell.StyleDefault.Bold(true)
	styleCenter  = styleDefault.Foreground(tcell.ColorOrangeRed)
	styleReverse = tcell.StyleDefault.Reverse(true).Bold(true)
)

// Display handles drawing our visualizer
type Display struct {
	barWidth int
	binWidth int

	screen tcell.Screen
	drawWg *sync.WaitGroup
}

// NewDisplay sets up the display
// should we panic or return an error as well?
// something to think about
func NewDisplay() *Display {

	screen, err := tcell.NewScreen()

	if err != nil {
		panic(err)
	}

	if err = screen.Init(); err != nil {
		panic(err)
	}

	screen.DisableMouse()
	screen.HideCursor()

	return &Display{
		barWidth: 2,
		binWidth: 3,
		screen:   screen,
		drawWg:   &sync.WaitGroup{},
	}
}

// Start display is bad
func (d *Display) Start(endCh chan<- bool) error {
	go func() {
		var ev tcell.Event
		for ev = d.screen.PollEvent(); ev != nil; ev = d.screen.PollEvent() {
			if d.HandleEvent(ev) != nil {
				break
			}
		}
		endCh <- true
	}()

	return nil
}

// HandleEvent will take events and do things with them
// TODO(noraih): MAKE THIS MORE ROBUST LIKE PREGO TOMATO SAUCE LEVELS OF ROBUST
func (d *Display) HandleEvent(ev tcell.Event) error {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return errors.New("rename this better please")
			default:
			}
		case tcell.KeyCtrlC:
			return errors.New("rename this better please")
		case tcell.KeyUp:
			d.setBarBin(d.barWidth+1, d.binWidth)
		case tcell.KeyRight:
			d.setBarBin(d.barWidth, d.binWidth+1)
		case tcell.KeyDown:
			d.setBarBin(d.barWidth-1, d.binWidth)
		case tcell.KeyLeft:
			d.setBarBin(d.barWidth, d.binWidth-1)
		default:

		}

	default:
	}

	return nil
}

// Stop display not work
func (d *Display) Stop() error {
	return nil
}

// Close will stop display and clean up the terminal
func (d *Display) Close() error {
	d.screen.Fini()
	return nil
}

func (d *Display) setBarBin(bar, bin int) {
	if bar < 1 {
		bar = 1
	}

	if bin < bar {
		if bin < 1 {
			bin = 0
		}
		bin += bar
	}

	d.barWidth = bar
	d.binWidth = bin
}

// SetWidths takes a bar width and spacing width
// Returns number of bars able to show
func (d *Display) SetWidths(bar, space int) int {
	if bar < 1 {
		bar = 1
	}

	if space < 0 {
		space = 0
	}

	d.barWidth = bar
	d.binWidth = bar + space

	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ int = d.screen.Size()
	return width / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var width, height int = d.screen.Size()
	return (width / d.binWidth), height
}

func (d *Display) offset() int {
	var width, _ int = d.screen.Size()
	width = width - (d.binWidth * (width / d.binWidth))
	if width > 1 {
		return width / 2
	}
	return 0
}

// temp for now

var drawDir = [...]int{-1, 1}

// Draw takes data, and draws
func (d *Display) Draw(height, delta int, sets ...*DataSet) error {

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	var cWidth = d.Bars() * d.binWidth

	// get our offset
	var cOffset = d.offset()

	for _, dSet := range sets {

		d.drawWg.Add(1)
		go drawSet(
			d.screen, dSet,
			height,
			d.barWidth, d.binWidth,
			cOffset, (delta * drawDir[dSet.ID()%len(drawDir)]),
			d.drawWg.Done,
		)
	}

	for xCol := 0; xCol < cWidth; xCol++ {
		if (xCol%d.binWidth)/d.barWidth > 0 {
			continue
		}

		// Draw our center line
		d.screen.SetContent(
			xCol+cOffset, height,
			DisplayBar, nil,
			styleCenter,
		)
	}

	d.drawWg.Wait()

	d.screen.Show()

	d.screen.Clear()

	return nil
}

func drawSet(s tcell.Screen, ds *DataSet, h, bw, fw, o, d int, fn func()) {

	for xBin, bin := range ds.Bins() {

		lCol := (xBin * fw) + o + bw
		lRow, vLast := toDrawVars(bin)

		for xCol, xRow := lCol-bw, 0; xCol < lCol; xCol++ {

			// we always want to target our bar height
			for xRow = 0; xRow < lRow; xRow++ {

				// Draw the bars for this data set
				s.SetContent(

					// TODO(nora): benchmark math (single loop) vs. double loop

					xCol,

					h+(d*(xRow+1)),

					// Just use our const character for now
					DisplayBar, nil,

					// Working on color bars
					styleDefault,
				)
			}

			if vLast > 0 {
				xRow++

				// Draw the bars for this data set
				if d < 0 {

					s.SetContent(
						xCol,

						h-xRow,

						// Just use our const character for now
						barHeightRunes[vLast], nil,

						// Working on color bars
						styleDefault,
					)
				} else {

					s.SetContent(
						xCol,

						h+(d*xRow),

						// Just use our const character for now
						barHeightRunes[numRunes-vLast], nil,

						// Working on color bars
						styleReverse,
					)
				}
			}
		}
	}

	fn()
}

func toDrawVars(a float64) (int, int) {
	whole, frac := math.Modf(math.Abs(a))
	frac = math.Max(0, 9*math.Min(0.9, frac))
	return int(whole), int(frac)
}
