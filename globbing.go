package main

import (
	"path/filepath"
	"time"
)

const globCheckInterval = 10 * time.Second

// Continually globs the given path checking for new files.
func GlobContinually(path string, apiKey string, fileConfigChan chan *FileConfig) error {
	logger.Infof("Discovering files for %s", path)

	globState := newGlobState(path, apiKey, fileConfigChan)

	// Perform an inital check, time.Ticket waits before it's first execution.
	err := globState.Check()
	if err != nil {
		return err
	}

	// Kick off the continual checking
	tick := time.Tick(globCheckInterval)
	return GlobWithTick(globState, tick)
}

// For testing purposes only.
func GlobWithTick(globState *globState, tick <-chan time.Time) error {
	for range tick {
		err := globState.Check()
		if err != nil {
			return err
		}
	}

	return nil
}

func newGlobState(path string, apiKey string, fileConfigChan chan *FileConfig) *globState {
	return &globState{
		path:           path,
		apiKey:         apiKey,
		currentPaths:   map[string]bool{},
		fileConfigChan: fileConfigChan,
		checkCount:     int64(0),
	}
}

type globState struct {
	path           string
	apiKey         string
	currentPaths   map[string]bool
	fileConfigChan chan *FileConfig
	checkCount     int64
}

// Performs a check on the path, sending new files to the fileConfig channel
func (g *globState) Check() error {
	paths, err := filepath.Glob(g.path)
	if err != nil {
		logger.Errorf("Error while globbling file path %s: %s", g.path, err)
		return err
	}

	if g.checkCount == 0 && len(g.currentPaths) == 0 && len(paths) == 0 {
		message := "File path %s did not return any files, the agent will continue checking " +
			"indefinitely. Please ensure the file(s) exist and that the Timber agent has permission to " +
			"access the file(s)."
		logger.Warnf(message, g.path)
	}

	for _, path := range paths {
		_, ok := g.currentPaths[path]
		if !ok {
			logger.Infof("Discovered new file from %s -> %s", g.path, path)

			g.currentPaths[path] = true
			newFileConfig := &FileConfig{
				Path:   path,
				ApiKey: g.apiKey,
			}
			g.fileConfigChan <- newFileConfig
		}
	}

	g.checkCount++

	return nil
}
