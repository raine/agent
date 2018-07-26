package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestFileTailer(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	tailer := NewFileTailer(file.Name(), false, true, nil, nil)
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
		if string(line.Lines) != <-expected {
			test.Fail()
		}
	}
}

func TestFileTailerListensOnStopChannel(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateFile, err := ioutil.TempFile("", "timber-agent-test-statefile.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	globalState.File = globalStateFile

	stop := make(chan bool, 1)

	tailer := NewFileTailer(file.Name(), false, true, nil, stop)
	stop <- true

	select {
	case _, open := <-tailer.Lines():
		if open {
			test.Fatal("tailer failed to close on stop")
		}
	// Wait up to 5 seconds for tailer to gracefully shutdown
	case <-time.After(5 * time.Second):
		test.Fatal("tailer failed to close within timeout after stop")
	}
}

func TestFileTailerPersistsState(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateFile, err := ioutil.TempFile("", "timber-agent-test-statefile.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	globalState.File = globalStateFile

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
	firstTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("one", 10))
	expectLines(test, firstTailer, generateLogLines("one", 10))

	quit <- true
	firstTailer.Wait()
	// Assume lines were sent successfully
	UpdateStateOffset(file.Name(), firstTailer.inner.Offset)

	sendLines(file, generateLogLines("two", 10))
	time.Sleep(5 * time.Millisecond)

	// with state file, start from previous spot
	secondTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("three", 10))
	expectLines(test, secondTailer, generateLogLines("two", 10))
	expectLines(test, secondTailer, generateLogLines("three", 10))

	quit <- true
	secondTailer.Wait()
	UpdateStateOffset(file.Name(), secondTailer.inner.Offset)

	sendLines(file, generateLogLines("four", 10))
	time.Sleep(5 * time.Millisecond)

	// after multiple runs, state file should contain most recent state
	thirdTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("five", 10))
	expectLines(test, thirdTailer, generateLogLines("four", 10))
	expectLines(test, thirdTailer, generateLogLines("five", 10))

	quit <- true
	thirdTailer.Wait()

	time.Sleep(5 * time.Millisecond)
	os.Remove(thirdTailer.filename)
}

func TestFileTailerIgnoresStateAfterRotation(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateFile, err := ioutil.TempFile("", "timber-agent-test-statefile.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	globalState.File = globalStateFile

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
	firstTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	sendLines(file, generateLogLines("one", 10))
	expectLines(test, firstTailer, generateLogLines("one", 10))

	quit <- true
	firstTailer.Wait()

	// rotate file
	file.Truncate(0)
	file.Seek(0, io.SeekStart)

	sendLines(file, generateLogLines("two", 10))
	time.Sleep(5 * time.Millisecond)

	// with state file that doesn't match, start from beginning
	secondTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("three", 10))
	expectLines(test, secondTailer, generateLogLines("two", 10))
	expectLines(test, secondTailer, generateLogLines("three", 10))

	quit <- true
	secondTailer.Wait()

	time.Sleep(5 * time.Millisecond)
	os.Remove(secondTailer.filename)
}

func TestFileTailerReadFromStart(test *testing.T) {
	file, err := ioutil.TempFile("", "timber-agent-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateFile, err := ioutil.TempFile("", "timber-agent-test-statefile.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	globalState.File = globalStateFile

	quit := make(chan bool)

	firstTailer := NewFileTailer(file.Name(), false, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	sendLines(file, generateLogLines("one", 10))
	expectLines(test, firstTailer, generateLogLines("one", 10))

	quit <- true
	firstTailer.Wait()

	secondTailer := NewFileTailer(file.Name(), true, true, quit, nil)
	time.Sleep(5 * time.Millisecond)

	go sendLines(file, generateLogLines("two", 10))
	expectLines(test, secondTailer, generateLogLines("one", 10))
	expectLines(test, secondTailer, generateLogLines("two", 10))
}

//cleanupFile Waits for the given file to be flushed to disk by the OS, and then removes it.
func cleanupFile(filepath string) {
	listen := make(chan error, 1)
	for {
		_, err := os.Stat(filepath)
		listen <- err
		select {
		case event := <-listen:
			if event == nil {
				os.Remove(filepath)
				return
			}
		case <-time.After(5 * time.Second):
			os.Remove(filepath)
			return
		}
	}
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
		case message := <-tailer.Lines():
			if string(message.Lines) != expectedLine {
				test.Fatalf("got '%s', expected '%s'", message.Lines, expectedLine)
			}
		case <-timeout:
			test.Fatalf("timed out expecting '%s'", expectedLine)
		}
	}
}

//
// Benchmarks
//

func BenchmarkTail(b *testing.B) {
	logger.Out = ioutil.Discard

	file, err := ioutil.TempFile("", "timber-agent-benchmark")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateFile, err := ioutil.TempFile("", "timber-agent-benchmark-statefile.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	globalState.File = globalStateFile

	tailer := NewFileTailer(file.Name(), false, true, nil, nil)

	for n := 0; n < b.N; n++ {
		sendLines(file, generateLogLines("benchmark line", 10000))

		for i := 10000; i > 0; i-- {
			<-tailer.Lines()
		}
	}
}
