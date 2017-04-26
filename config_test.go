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
