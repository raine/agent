package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGlobbingDiscoversNewFiles(test *testing.T) {
	// Setup the directory
	testFilesDirPath, _ := filepath.Abs("test/tmp/globbing_test_files")
	os.RemoveAll(testFilesDirPath)
	os.MkdirAll(testFilesDirPath, os.ModePerm)

	globFilePath := fmt.Sprintf("%s/*.log", testFilesDirPath)
	apiKey := "apikey"
	fileConfigsChan := make(chan *FileConfig)
	globState := newGlobState(globFilePath, apiKey, fileConfigsChan)
	tick := make(chan time.Time)


	go func() {
		err := GlobWithTick(globState, tick)
		if err != nil {
			test.Fatal(err)
		}
	}()

	// Add the first file
	firstFilePath := fmt.Sprintf("%s/first.log", testFilesDirPath)
	_, err := os.Create(firstFilePath)
	if err != nil {
		test.Fatal(err)
	}

	// Test the first tick
	tick <- time.Now()
	firstFileConfig := <-fileConfigsChan
	if firstFileConfig.Path != firstFilePath {
		test.Fatalf("Expected to receive file %s but got %s", firstFilePath, firstFileConfig.Path)
	}

	// Add the second file
	secondFilePath := fmt.Sprintf("%s/second.log", testFilesDirPath)
	_, err = os.Create(secondFilePath)
	if err != nil {
		test.Fatal(err)
	}

	// Test the second tick
	tick <- time.Now()
	secondFileConfig := <-fileConfigsChan
	if secondFileConfig.Path != secondFilePath {
		test.Fatalf("Expected to receive file %s but got %s", secondFilePath, secondFileConfig.Path)
	}

	// Cleanup
	os.RemoveAll(testFilesDirPath)
}
