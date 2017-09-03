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

	if config.BatchPeriodSeconds != 10 {
		t.Error("batch period was not defaulted")
	}
}

func TestParseConfigSpecificApiKey(t *testing.T) {
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

func TestParseConfigDefaultApiKey(t *testing.T) {
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

func TestnormalizeConfigDefaultFileApiKey(t *testing.T) {
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

func TestnormalizeConfigNoApiKey(t *testing.T) {
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
