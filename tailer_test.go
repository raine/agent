package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

var logger = log.New(os.Stderr, "", 0)

func TestFileTailer(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	tailer := NewFileTailer(file.Name(), true, nil, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("test", 100))
	expectLines(test, tailer, generateLogLines("test", 100))
}

func TestReaderTailer(test *testing.T) {
	var buf bytes.Buffer
	for line := range generateLogLines("test", 10) {
		buf.WriteString(line + "\n")
	}

	tailer := NewReaderTailer(&buf, nil)

	expected := generateLogLines("test", 10)
	for line := range tailer.Lines() {
		if line != <-expected {
			test.Fail()
		}
	}
}

func TestFileTailerPersistsState(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	// Writes 256 bytes to the file as initial data; this ensures that
	// the file hash can be computed properly
	for i := 0; i < 256; i++ {
		if _, err = file.WriteString("-"); err != nil {
			panic(err)
		}
	}

	file.WriteString("\n")

	fmt.Fprintln(file, "skip me")
	quit := make(chan bool)

	// with no state file, tail should start at the end
	firstTailer := NewFileTailer(file.Name(), true, quit, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("one", 10))
	expectLines(test, firstTailer, generateLogLines("one", 10))

	quit <- true
	firstTailer.Wait()

	sendLines(file, generateLogLines("two", 10))
	time.Sleep(5 * time.Millisecond)

	// with state file, start from previous spot
	secondTailer := NewFileTailer(file.Name(), true, quit, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("three", 10))
	expectLines(test, secondTailer, generateLogLines("two", 10))
	expectLines(test, secondTailer, generateLogLines("three", 10))

	quit <- true
	secondTailer.Wait()

	sendLines(file, generateLogLines("four", 10))
	time.Sleep(5 * time.Millisecond)

	// after multiple runs, state file should contain most recent state
	thirdTailer := NewFileTailer(file.Name(), true, quit, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("five", 10))
	expectLines(test, thirdTailer, generateLogLines("four", 10))
	expectLines(test, thirdTailer, generateLogLines("five", 10))

	quit <- true
	thirdTailer.Wait()

	time.Sleep(5 * time.Millisecond)
	thirdTailer.RemoveStatefile()
}

func TestFileTailerIgnoresStateAfterRotation(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	// Writes 256 bytes to the file as initial data; this ensures that
	// the file hash can be computed properly
	for i := 0; i < 256; i++ {
		if _, err = file.WriteString("-"); err != nil {
			panic(err)
		}
	}

	file.WriteString("\n")

	fmt.Fprintln(file, "skip me")
	quit := make(chan bool)

	// with no state file, tail should start at the end
	firstTailer := NewFileTailer(file.Name(), true, quit, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("one", 10))
	expectLines(test, firstTailer, generateLogLines("one", 10))

	quit <- true
	firstTailer.Wait()

	// rotate file
	file.Truncate(0)
	file.Seek(0, io.SeekStart)

	sendLines(file, generateLogLines("two", 10))
	time.Sleep(5 * time.Millisecond)

	// with state file that doesn't match, start from beginning
	secondTailer := NewFileTailer(file.Name(), true, quit, logger)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("three", 10))
	expectLines(test, secondTailer, generateLogLines("two", 10))
	expectLines(test, secondTailer, generateLogLines("three", 10))

	quit <- true
	secondTailer.Wait()

	time.Sleep(5 * time.Millisecond)
	secondTailer.RemoveStatefile()
}

func generateLogLines(prefix string, n int) chan string {
	ch := make(chan string)

	go func() {
		for i := 0; i < n; i++ {
			ch <- fmt.Sprintf("%s %d", prefix, i)
		}
		close(ch)
	}()

	return ch
}

func sendLines(w io.Writer, lines chan string) {
	for line := range lines {
		fmt.Fprintln(w, line)
	}
}

func expectLines(test *testing.T, tailer Tailer, lines chan string) {
	timeout := time.After(5 * time.Second)
	for expectedLine := range lines {
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
