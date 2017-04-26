package main

import (
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

func TestBasicTailing(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	tailer := NewFileTailer(file.Name(), false, nil)

	go func() {
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
