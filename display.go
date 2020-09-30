package tavis

import (
	"github.com/gdamore/tcell/v2"
)

// DisplayBar is the block we use for bars
const (
	DisplayBar   rune = '\u2588'
	DisplaySpace rune = '\u0020'

	MaxWidth = 5000
)

// Display handles drawing our visualizer
type Display struct {
	screen   tcell.Screen
	DataSets []*DataSet
	barWidth int
	binWidth int
}

// Init sets up the display
func (d *Display) Init() error {
	var err error

	// cellBuf = &tcell.CellBuffer{}

	if d.screen, err = tcell.NewScreen(); err != nil {
		return err
	}

	if err = d.screen.Init(); err != nil {
		return err
	}

	d.screen.DisableMouse()
	d.screen.HideCursor()

	d.SetWidths(1, 1)

	return nil
}

// Start display is bad
func (d *Display) Start(endCh chan<- bool) error {
	go func() {
		var ev tcell.Event
		for ev = d.screen.PollEvent(); ev != nil; ev = d.screen.PollEvent() {
			if d.HandleEvent(ev) {
				break
			}
		}
		endCh <- true
	}()

	return nil
}

// HandleEvent will take events and do things with them
func (d *Display) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlC:
			return true
		default:

		}

	default:
	}

	return false
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

// SetWidths takes a bar width and spacing width
// Returns number of bars able to show
func (d *Display) SetWidths(bar, space int) int {
	d.barWidth = bar
	d.binWidth = (bar + space)

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

// Draw takes data, and draws
func (d *Display) Draw() error {
	var (
		cHeight int
		cWidth  int
		cOffset int

		xCol int
		xRow int
		xBin int

		vSet    *DataSet
		vTarget int
		vDelta  int
	)

	cWidth, cHeight = d.screen.Size()

	cHeight = cHeight / 2

	cOffset = d.offset()

	for _, vSet = range d.DataSets {

		vDelta = 1

		if vSet.id == 1 {
			vDelta = -vDelta
		}

		for xCol = 0; xCol < cWidth; xCol += d.binWidth {

			xBin = xCol / d.binWidth

			vTarget = int(vSet.Data[xBin])

			for xRow = 0; xRow < vTarget*d.barWidth; xRow++ {
				d.screen.SetContent(
					xCol+cOffset+(xRow/vTarget),
					cHeight+(vDelta*(xRow%vTarget)),
					DisplayBar, nil,
					tcell.StyleDefault,
				)
			}
		}
	}

	d.screen.Show()
	d.screen.Clear()

	return nil
}
