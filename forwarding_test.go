package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/timberio/agent/test/server"
)

func TestForwarding(test *testing.T) {
	bufChan := make(chan *bytes.Buffer, 1)
	bufChan <- bytes.NewBufferString("test log line\n")
	close(bufChan)

	var output bytes.Buffer
	go server.AcceptLogs(&output)

	Forward(bufChan, http.DefaultTransport, "http://localhost:8080/frames", "api key")

	actual := strings.TrimSpace(output.String())
	expected := "test log line"

	if actual != expected {
		test.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
	}
}
