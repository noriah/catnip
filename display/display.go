package display

import (
	"errors"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/noriah/tavis/dsp"
)

const (

	// MaxWidth will be removed at some point
	MaxWidth = 5000
)

// bar blocks for later
var (
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

// New sets up the display
// should we panic or return an error as well?
// something to think about
func New() *Display {

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
	var width, _ = d.screen.Size()
	return width / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var width, height = d.screen.Size()
	return (width / d.binWidth), height
}

// Draw takes data, and draws
func (d *Display) Draw(height, delta int, sets ...*dsp.DataSet) error {

	// get our offset
	var cOffset = d.Bars() - 1
	cOffset *= d.binWidth
	cOffset += d.barWidth

	var cWidth, _ = d.screen.Size()

	cWidth, cOffset = cOffset, cWidth-cOffset
	cOffset /= 2
	// cWidth -= cOffset

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	for _, dSet := range sets {

		d.drawWg.Add(1)
		go drawBars(d, dSet, height, cOffset, delta)
	}

	for xCol, xBin := 0, 0; xCol < cWidth; xCol = xBin * d.binWidth {
		for lCol := xCol + d.barWidth; xCol < lCol; xCol++ {
			// Draw our center line
			d.screen.SetContent(
				cOffset+xCol, height,
				DisplayBar, nil,
				styleCenter,
			)
		}
		xBin++
	}

	d.drawWg.Wait()

	d.screen.Show()

	d.screen.Clear()

	return nil
}
