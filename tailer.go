package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func freshBuffer() *bytes.Buffer {
	// preallocate 2MB
	buf := bytes.NewBuffer(make([]byte, 2e6))
	buf.Reset()
	return buf
}

type Tailer struct {
	Lines  chan string
	ApiKey string
	After  func()

	Buf *bytes.Buffer

	// we send "finished" buffers over this channel to be sent as http requests
	// asynchonously. a slowdown in the sender goroutine will eventually (based on
	// channel buffering) provide backpressure on the log tailing goroutine, which
	// should shed load in response.
	//
	// this design relies on the gc to clean up old buffers, but an alternative
	// would be to have a second channel for sending back old buffers for reuse,
	// which could be a good option if we're seeing excess memory pressure
	BufChan chan *bytes.Buffer
}

func NewTailer(lines chan string, apiKey string) Tailer {
	return Tailer{
		Lines:   lines,
		ApiKey:  apiKey,
		After:   func() {},
		Buf:     freshBuffer(),
		BufChan: make(chan *bytes.Buffer),
	}
}

func (t *Tailer) Run(config *Config, quit chan bool) error {
	token := base64.StdEncoding.EncodeToString([]byte(t.ApiKey))

	// we use this channel to wait until the sender has finished before exiting
	done := make(chan bool)

	go sender(config.Endpoint, token, t.BufChan, done)

	tick := time.Tick(time.Duration(config.BatchPeriodSeconds) * time.Second)
	for {
		select {
		case line, ok := <-t.Lines:
			if t.Buf.Len()+len(line)+1 > t.Buf.Cap() {
				t.flush()
			}
			if len(line) > 0 {
				io.WriteString(t.Buf, line+"\n")
			}
			if !ok { // channel is closed
				t.flush()
				t.stop(done)
				return nil
			}
		case <-tick:
			if t.Buf.Len() > 0 {
				t.flush()
			}
		case <-quit:
			t.flush()
			t.stop(done)
			return nil
		}
	}
}

func (t *Tailer) flush() {
	if t.Buf.Len() > 0 {
		t.BufChan <- t.Buf
		t.Buf = freshBuffer()
	}
}

func (t *Tailer) stop(done chan bool) {
	close(t.BufChan)
	<-done
	t.After()
}

func sender(endpoint, token string, ch chan *bytes.Buffer, done chan bool) {
	transport := http.DefaultTransport
	for buf := range ch {
		req, err := http.NewRequest("POST", endpoint, buf)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
		req.Header.Add("Accept", "application/json")
		resp, err := transport.RoundTrip(req)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("flushed buffer, got status code %d", resp.StatusCode)
			resp.Body.Close()
		}
	}
	done <- true
}
