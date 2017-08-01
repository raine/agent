// Implements Daemonize() for Linux platforms.
//
// See daemon.go for an extended discussion of why this functionality
// is only implemented for Linux.

package main

import (
	"github.com/VividCortex/godaemon"
)

// Daemonize will attempt to daemonize the executable by calling
// the godaemon package to perform a non-native forking operation; traditionally
// this would be accomplished using the appropriate OS level functionality,
// however this is not possible in Go. The call to `godaemon.MakeDaemon` will
// cause the currently running agent to exit and launch a new agent outside
// the current context.
//
// If daemonization fails, the error is returned to the caller and the caller is
// expected to log the error message and exit.
func Daemonize() error {
	_, _, err := godaemon.MakeDaemon(&godaemon.DaemonAttr{})

	return err
}
