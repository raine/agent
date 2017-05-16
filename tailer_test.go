package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func generateLogLines(n int) chan string {
	ch := make(chan string)

	go func() {
		for i := 0; i < n; i++ {
			ch <- fmt.Sprintf("test log line %d", i)
		}
		close(ch)
	}()

	return ch
}

func TestFileTailer(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	tailer := NewFileTailer(file.Name(), false, nil)

	go func() {
		time.Sleep(5 * time.Millisecond)
		for line := range generateLogLines(100) {
			fmt.Fprintln(file, line)
		}
	}()

	timeout := time.After(100 * time.Millisecond)
	for expectedLine := range generateLogLines(100) {
		select {
		case line := <-tailer.Lines():
			if line != expectedLine {
				test.Fatalf("got '%s', expected '%s'", line, expectedLine)
			}
		case <-timeout:
			test.Fatalf("timed out expecting '%s'", expectedLine)
		}
	}
}

func TestReaderTailer(test *testing.T) {
	var buf bytes.Buffer
	for line := range generateLogLines(10) {
		buf.WriteString(line + "\n")
	}

	tailer := NewReaderTailer(&buf, nil)

	expected := generateLogLines(10)
	for line := range tailer.Lines() {
		if line != <-expected {
			test.Fail()
		}
	}
}

func TestFileTailerStartsAtEnd(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	fmt.Fprintln(file, "skip me")

	tailer := NewFileTailer(file.Name(), false, nil)

	go func() {
		time.Sleep(5 * time.Millisecond)
		for line := range generateLogLines(100) {
			fmt.Fprintln(file, line)
		}
	}()

	timeout := time.After(100 * time.Millisecond)
	for expectedLine := range generateLogLines(100) {
		select {
		case line := <-tailer.Lines():
			if line != expectedLine {
				test.Fatalf("got '%s', expected '%s'", line, expectedLine)
			}
		case <-timeout:
			test.Fatalf("timed out expecting '%s'", expectedLine)
		}
	}
}
