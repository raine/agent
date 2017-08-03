package main

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	empty := strings.NewReader("")

	config, err := readConfig(empty)
	if err != nil {
		panic(err)
	}

	if config.Endpoint != "https://logs.timber.io/frames" {
		t.Error("endpoint was not defaulted")
	}

	if config.BatchPeriodSeconds != 10 {
		t.Error("batch period was not defaulted")
	}
}

func TestApiKey(t *testing.T) {
	configString := `
[[files]]
path = "/var/log/log.log"
api_key = "abc:1234"
`

	configFile := strings.NewReader(configString)

	config, err := readConfig(configFile)

	if err != nil {
		panic(err)
	}

	expectedFileCount := 1
	fileCount := len(config.Files)

	if fileCount != expectedFileCount {
		t.Fatalf("Expected %d files from configuration but %d reported", expectedFileCount, fileCount)
	}

	expectedApiKey := "abc:1234"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}
