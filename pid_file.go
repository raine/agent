package main

import (
	"os"
	"strconv"
)

// writePIDFile writes the agent's process ID to the given file location
func writePIDFile(pidfileLocation string) error {
	pid := int64(os.Getpid())
	pidString := strconv.FormatInt(pid, 10)

	pidfile, err := os.OpenFile(pidfileLocation, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		logger.Errorf("Failed to open \"%s\" to write PID to: %v\n", pidfileLocation, err)
		return err
	}

	defer pidfile.Close()

	_, err = pidfile.WriteString(pidString)
	if err != nil {
		logger.Errorf("Failed to write PID to open file \"%s\": %v\n", pidfileLocation, err)
		return err
	}

	logger.Infof("PID file written to %s", pidfileLocation)

	return nil
}

// removePIDFile deletes the file holding the process ID
func removePIDFile(pidfileLocation string) error {
	err := os.Remove(pidfileLocation)

	if err != nil {
		logger.Errorf("Unable to remove PID file from \"%s\": %v", pidfileLocation, err)
		return err
	}

	return nil
}
