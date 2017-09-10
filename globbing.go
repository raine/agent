package main

import (
	"path/filepath"
	"time"
)

const globCheckInterval = 10 * time.Second

func Glob(fileConfigChan chan *FileConfig, fileConfig *FileConfig) error {
	tick := time.Tick(globCheckInterval)
	return GlobWithTick(tick, fileConfigChan, fileConfig)
}

func GlobWithTick(tick <-chan time.Time, fileConfigChan chan *FileConfig, fileConfig *FileConfig) error {
	currentPaths := map[string]bool{}

	logger.Info("Discovering files for %s", fileConfig.Path)

	for range tick {
		paths, err := filepath.Glob(fileConfig.Path)
		if err != nil {
			logger.Errorf("Error while globbling file path %s: %s", fileConfig.Path, err)
			return err
		}

		if len(currentPaths) == 0 && len(paths) == 0 {
			logger.Warnf("File path %s did not discover _any_ files, the agent will check again momentarily", fileConfig.Path)
		}

		for _, path := range paths {
			_, ok := currentPaths[path]
			if !ok {
				logger.Infof("Disovered new file from %s -> %s", fileConfig.Path, path)

				currentPaths[path] = true
				newFileConfig := &FileConfig{
					Path:   path,
					ApiKey: fileConfig.ApiKey,
				}
				fileConfigChan <- newFileConfig
			}
		}
	}

	return nil
}
