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

			if buf.Len()+len(line)+1 > buf.Cap() {
				bufChan <- buf
				buf = freshBuffer()
			}

			if len(line) > 0 {
				io.WriteString(buf, line+"\n")
			}

			if !ok { // channel is closed
				if buf.Len() > 0 {
					bufChan <- buf
					buf = freshBuffer()
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
	// preallocate 2MB
	buf := bytes.NewBuffer(make([]byte, 2e6))
	buf.Reset()
	return buf
}
