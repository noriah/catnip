package tavis

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// constants for testing
const (
	DeviceName   = "VisOut"
	SampleRate   = 48000
	TargetFPS    = 60
	ChannelCount = 2

	SignalChanSize = 3
)

// calculated constants
const (
	SampleSize = int(SampleRate) / TargetFPS
	BufferSize = SampleSize * ChannelCount
	DrawDelay  = time.Second / TargetFPS
)

// Run does the run things
func Run() error {

	var (
		drawTicker *time.Ticker
		rootCtx    context.Context
		rootCancel context.CancelFunc

		signalCtx    context.Context
		signalCancel context.CancelFunc
	)

	rootCtx, rootCancel = context.WithCancel(context.Background())
	defer rootCancel()

	signalCtx, signalCancel = context.WithCancel(rootCtx)
	defer signalCancel()

	drawTicker = time.NewTicker(DrawDelay)

	go signalTown(signalCtx, rootCancel)

RunForRest: // run!!!
	for {
		select {
		case <-rootCtx.Done():
			break RunForRest
		case <-drawTicker.C:
		}

		draw()
	}

	return nil
}

var idx = 0

func draw() {
	idx++
	fmt.Println("hithere", idx)
}

// lets take a ride on down to flav--imean signal town!
func signalTown(ctx context.Context, cancel context.CancelFunc) {

	// [crys]: it is really more of a junction.
	var signalJunction chan os.Signal

	// [nora]: oh. but what about all the signals we will see?
	// there has to be at least... ones of them, right?
	signalJunction = make(chan os.Signal, SignalChanSize)

	// [crys]: it is a junction for two signals.
	signal.Notify(signalJunction, syscall.SIGINT, syscall.SIGTERM)

	// [winter]: well what do they do? if we are are taking the time to write
	// this function separate, they must be important right?
	var recv os.Signal

TheSignalTownShuffle:
	for {
		select {
		case <-ctx.Done():
			break TheSignalTownShuffle
		case recv = <-signalJunction:
			switch recv {
			// [nix]: please stop, nora. i want to get this done.
			case syscall.SIGINT, syscall.SIGTERM:
				break TheSignalTownShuffle
			default:
			}
		}
	}

	// [nora,crys]: aww. why are you so down tonight, nix?
	signal.Stop(signalJunction)

	// [nix]: i feel so unproductive tonight.
	close(signalJunction)

	cancel()
	fmt.Println("signal function ended")
}
