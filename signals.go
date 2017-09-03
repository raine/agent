package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Amount of time other go routines have to exit after an
// OS exit signal is received from the OS before the agent
// forcibly exits
const signalHandlingTimeout = 5 * time.Second

// handleSignals returns a chan bool that will be closed
// when an OS signal is sent requesting the agent to shut down.
//
// If the request to shut down is not handled in a certain amount of time
// or the OS sends another shut down signal, the agent will immediately
// exit
func handleSignals() chan bool {
	quit := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Starts a goroutine that will handle the inbound signal
	// from the OS. When a signal is received, the quit channel
	// gets closed, which signals to the tailing library to stop.
	go func() {
		signal := <-signals
		logger.Infof("got %s, shutting down...", signal)
		close(quit)
		timeout := time.After(signalHandlingTimeout)
		select {
		case <-signals:
			os.Exit(1)
		case <-timeout:
			os.Exit(1)
		}
	}()

	return quit
}
