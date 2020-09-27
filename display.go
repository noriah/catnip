package tavis

import (
	"os"

	"github.com/gdamore/tcell/v2"
)

// DisplayBar is the block we use for bars
const (
	DisplayBar   rune = '\u2588'
	DisplaySpace rune = '\u0020'
)

// Display handles drawing our visualizer
type Display struct {
	screen     tcell.Screen
	barWidth   int
	spaceWidth int

	shouldRun bool
}

// Init sets up the display
func (d *Display) Init() error {
	var err error

	if d.screen, err = tcell.NewScreen(); err != nil {
		return err
	}

	if err = d.screen.Init(); err != nil {
		return err
	}

	d.SetWidths(2, 1)

	return nil
}

func (d *Display) Start() error {

	d.shouldRun = true

	go func() {
		for d.shouldRun {

			if event := d.screen.PollEvent(); event != nil {
				os.Interrupt.Signal()
			}
		}
	}()

	return nil
}

func (d *Display) Stop() error {
	d.shouldRun = false
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
	d.spaceWidth = space

	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ int = d.screen.Size()
	var perBar int = d.barWidth + d.spaceWidth

	width = width - d.spaceWidth
	width = width / perBar

	return width
}

// Size returns the width and height of the screen
func (d *Display) Size() (int, int) {
	var _, height int = d.screen.Size()
	var width = d.Bars()
	return width, height
}

// Draw takes data, and draws
func (d *Display) Draw(buf []float64) error {
	var (
		totalHeight int
		totalBars   int

		centerHeight int

		barSpaceWidth int

		bax int
		lix int
	)

	totalBars, totalHeight = d.Size()

	barSpaceWidth = d.barWidth + d.spaceWidth

	centerHeight = totalHeight / 2

	for bax = 0; bax < totalBars; bax++ {

		for lix = 0; lix < centerHeight; lix++ {
			d.screen.SetCell(bax*barSpaceWidth, lix, tcell.StyleDefault, DisplayBar)
		}
	}

	d.screen.Sync()

	return nil
}
