package main

import (
	"bytes"
	"io"
	"time"
)

func Batch(lines chan string, bufChan chan *bytes.Buffer, batchPeriodSeconds int64) {
	buf := freshBuffer()
	tick := time.Tick(time.Duration(batchPeriodSeconds) * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if ok {
				if len(line)+1 > buf.Cap() {
					logger.Warn("Ignoring log line greater than the max payload size (1 MB)")
					continue
				}

				if buf.Len()+len(line)+1 > buf.Cap() {
					bufChan <- buf
					buf = freshBuffer()
				}

				if len(line) > 0 {
					io.WriteString(buf, line+"\n")
				}

			} else { // channel is closed
				if buf.Len() > 0 {
					bufChan <- buf
				}
				close(bufChan)
				return
			}

		case <-tick:
			if buf.Len() > 0 {
				bufChan <- buf
				buf = freshBuffer()
			}
		}
	}
}

func freshBuffer() *bytes.Buffer {
	// Preallocate 990kb. The Timber API will not accept payloads larger than 1mb.
	// This leaves 10kb for headers.
	buf := bytes.NewBuffer(make([]byte, 990000))
	buf.Reset()
	return buf
}
