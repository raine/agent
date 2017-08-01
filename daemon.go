// Implements Daemonize() for non-Linux platforms.
//
// We originally introduced the daemonization option as a crutch for Amazon
// Elasticbeanstalk since the Amazon Linux version it utilizes by default
// comes with no native daemonization tooling. In order to provide
// daemonization, we rely on the godaemon package which fakes the
// standard daemonization process; while it is not true daemonization,
// it works well enough.
//
// The `godaemon` package has per-OS daemonization techniques.
//
//  - The Linux variant works by mimic'ing the standard demonization process
//  using calls to Linux filesystem descriptors and then using pure Go
//  (i.e., not Cgo) function calls
//  - Both the Darwin and FreeBSD variants depend on C library functions
//  specific to the operating system
//  - The Windows variant depends on a DLL being loaded
//
// The Darwin, FreeBSD, and Windows variants introduce complexities when
// performing cross-platform builds. Since the build system is not guaranteed
// to have a compatible version of the C library (or in the Windows, the
// relevant DLL), builds will fail.
//
// Since daemonization is only needed for Linux, we only build support for
// daemonization with Linux builds. If a user attempts to daemonize on a
// different platform, the utility will exit with a notice to use a native
// daemonization tool.

// +build !linux

package main

import (
	"errors"
)

// Daemonize will return an error indicatin that daemonization is not
// possible on this platform.
func Daemonize() error {
	errText := "Daemonization is not possible on this platform"
	return errors.New(errText)
}
