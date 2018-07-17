package main

import (
	"bytes"
	"testing"
)

func TestChannelClosing(t *testing.T) {
	lines := make(chan *LogMessage)
	bufChan := make(chan *LogMessage)

	go Batch(lines, bufChan, 10)
	lines <- &LogMessage{Lines: []byte("test log line")}
	close(lines)

	actual := <-bufChan
	expected := "test log line\n"
	if string(actual.Lines) != expected {
		t.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}

func TestBufferOverflow(t *testing.T) {
	lines := make(chan *LogMessage)
	bufChan := make(chan *LogMessage)

	go Batch(lines, bufChan, 10)
	filler := "test log line"
	fillerLen := len(filler) + 1
	for written := 0; written+fillerLen < 990000; written += fillerLen {
		lines <- &LogMessage{Lines: []byte(filler)}
	}
	lines <- &LogMessage{Lines: []byte("overflowed")}
	close(lines)

	<-bufChan // throw away the big one
	actual := <-bufChan
	expected := "overflowed\n"
	if string(actual.Lines) != expected {
		t.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}

// Batch()
// Log lines larger than the max payload size (1 MB) should be dropped
func TestBatchDropLogLine(t *testing.T) {
	lines := make(chan *LogMessage)
	bufChan := make(chan *LogMessage)

	filler := "test log line"
	buf := bytes.NewBuffer(make([]byte, 990000))
	for buf.Len() < 990000 {
		buf.WriteString(filler)
	}
	logline := buf.String()

	go Batch(lines, bufChan, 10)
	lines <- &LogMessage{Lines: []byte(logline)}
	close(lines)

	// Nothing should be sent to bufChan since we are dropping message
	actual := <-bufChan
	if actual != nil {
		t.Fatalf("expected \"%+v\" to be nil", actual)
	}
}
