package main

import (
	"bytes"
	"testing"
)

func TestChannelClosing(t *testing.T) {
	lines := make(chan string)
	bufChan := make(chan *bytes.Buffer)

	go Batch(lines, bufChan, 10)
	lines <- "test log line"
	close(lines)

	actual := <-bufChan
	expected := "test log line\n"
	if actual.String() != expected {
		t.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}

func TestBufferOverflow(t *testing.T) {
	lines := make(chan string)
	bufChan := make(chan *bytes.Buffer)

	go Batch(lines, bufChan, 10)
	filler := "test log line"
	fillerLen := len(filler) + 1
	for written := 0; written+fillerLen < 2e6; written += fillerLen {
		lines <- filler
	}
	lines <- "overflowed"
	close(lines)

	<-bufChan // throw away the big one
	actual := <-bufChan
	expected := "overflowed\n"
	if actual.String() != expected {
		t.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}
