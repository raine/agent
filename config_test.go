package main

import (
	"strings"
	"testing"
)

func TestNewConfigSetsDefaults(t *testing.T) {
	config := NewConfig()

	if config.Endpoint != "https://logs.timber.io/frames" {
		t.Error("endpoint was not defaulted")
	}

	if config.BatchPeriodSeconds != 3 {
		t.Error("batch period was not defaulted")
	}
}

func TestNewConfigSpecificApiKey(t *testing.T) {
	configString := `
[[files]]
path = "/var/log/log.log"
api_key = "abc:1234"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
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

func TestNewConfigDefaultApiKey(t *testing.T) {
	configString := `
default_api_key = "zyx:0987"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedDefaultApiKey := "zyx:0987"
	defaultApiKey := config.DefaultApiKey

	if defaultApiKey != expectedDefaultApiKey {
		t.Errorf("Expected DefaultApiKey to be %s but got %s", expectedDefaultApiKey, defaultApiKey)
	}
}

func TestNewConfigDefaultFileApiKey(t *testing.T) {
	configString := `
default_api_key = "zyx:0987"

[[files]]
path = "/var/log/log.log"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedApiKey := "zyx:0987"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewConfigNoApiKey(t *testing.T) {
	configString := `
[[files]]
path = "/var/log/log.log"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedApiKey := ""
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewConfigMultipleFiles(t *testing.T) {
	configString := `
default_api_key = "default_api_key"

[[files]]
path = "/var/log/log1.log"
api_key = "file1_api_key"

[[files]]
path = "/var/log/log2.log"
`

	config := NewConfig()
	configFile := strings.NewReader(configString)
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	// Check the file count
	expectedFileCount := 2
	fileCount := len(config.Files)

	if fileCount != expectedFileCount {
		t.Fatalf("Expected %d files from configuration but %d reported", expectedFileCount, fileCount)
	}

	// Check the first file path
	expectedPath := "/var/log/log1.log"
	path := config.Files[0].Path

	if path != expectedPath {
		t.Errorf("Expected Path to be %s but got %s", expectedPath, path)
	}

	// Check the first file api key
	expectedApiKey := "file1_api_key"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}

	// Check the second file path
	expectedPath = "/var/log/log2.log"
	path = config.Files[1].Path

	if path != expectedPath {
		t.Errorf("Expected Path to be %s but got %s", expectedPath, path)
	}

	// Check the second file api key
	expectedApiKey = "default_api_key"
	apiKey = config.Files[1].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}
