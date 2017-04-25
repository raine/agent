package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/tail"
	"github.com/timberio/agent/test/server"
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

	t, err := tail.TailFile(file.Name(), tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		panic(err)
	}

	go func() {
		for line := range generateLogLines(100) {
			fmt.Fprintln(file, line)
		}
	}()

	timeout := time.After(100 * time.Millisecond)
	for expectedLine := range generateLogLines(100) {
		select {
		case tailedLine := <-t.Lines:
			if tailedLine.Text != expectedLine {
				test.Fatalf("got '%s', expected '%s'", tailedLine.Text, expectedLine)
			}
		case <-timeout:
			test.Fatalf("timed out expecting '%s'", expectedLine)
		}
	}
}

func TestForwarding(test *testing.T) {
	var output bytes.Buffer
	go server.AcceptLogs(&output)

	tailer := NewTailer(generateLogLines(5), "api key")
	tailer.Run(&Config{Endpoint: "http://localhost:8080/frames"}, nil)

	actual := strings.TrimSpace(output.String())
	expected := `test log line 0
test log line 1
test log line 2
test log line 3
test log line 4`

	if actual != expected {
		test.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}
