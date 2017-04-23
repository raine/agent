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

type Tailer struct {
	Lines  chan string
	ApiKey string
	After  func()
}

func (t *Tailer) Run(config *Config, quit chan bool) error {
	token := base64.StdEncoding.EncodeToString([]byte(t.ApiKey))

	// we send "finished" buffers over this channel to be sent as http requests
	// asynchonously. a slowdown in the sender goroutine will eventually (based on
	// channel buffering) provide backpressure on the log tailing goroutine, which
	// should shed load in response.
	//
	// this design relies on the gc to clean up old buffers, but an alternative
	// would be to have a second channel for sending back old buffers for reuse,
	// which could be a good option if we're seeing excess memory pressure
	done := make(chan bool)
	bufChan := make(chan *bytes.Buffer)
	go sender(config.Endpoint, token, bufChan, done)

	buf := bytes.NewBuffer([]byte{})
	tick := time.Tick(time.Duration(config.BatchPeriodSeconds) * time.Second)
	for {
		select {
		case line, ok := <-t.Lines:
			// TODO: check len + len before doing this
			io.WriteString(buf, line+"\n")
			// TODO: make this configurable
			if buf.Len() > 1000000 {
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
			if !ok { // channel is closed
				bufChan <- buf
				close(bufChan)
				// wait for sender to finish
				<-done
				t.After()
				return nil
			}
		case <-tick:
			if buf.Len() > 0 {
				// TODO: extract a shared version of this, maybe preallocate 2MB buffers
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
		case <-quit:
			bufChan <- buf
			close(bufChan)
			// wait for sender to finish
			<-done
			t.After()
			return nil
		}
	}
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
