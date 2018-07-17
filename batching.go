package main

import (
	"bytes"
	"time"
)

func Batch(messages chan *LogMessage, batchChan chan *LogMessage, batchPeriodSeconds int64) {
	// As *LogMessage are read from the messages channel and added to our internal buffer, we also store the message's
	// position and filename. This is so that when we flush the buffer, we can also send the source file of the buffer's
	// contents as well as our posititon in the file. These values are written to the agent's globalState in order to
	// resume where we left off on agent restart.
	var position int64
	var filename string

	buf := freshBuffer()
	tick := time.Tick(time.Duration(batchPeriodSeconds) * time.Second)
	for {
		select {
		case message, ok := <-messages:
			if ok {
				line := message.Lines
				filename = message.Filename

				if len(line)+1 > buf.Cap() {
					logger.Warn("Ignoring log line greater than the max payload size (1 MB)")
					continue
				}

				if buf.Len()+len(line)+1 > buf.Cap() {
					newMessage := &LogMessage{
						Filename: filename,
						Lines:    buf.Bytes(),
						Position: position,
					}

					batchChan <- newMessage
					buf = freshBuffer()
				}

				if len(line) > 0 {
					buf.Write(append(line, "\n"...))

					filename = message.Filename
					position = message.Position
				}

			} else { // channel is closed
				if buf.Len() > 0 {
					newMessage := &LogMessage{
						Filename: filename,
						Lines:    buf.Bytes(),
						Position: position,
					}

					batchChan <- newMessage
				}
				close(batchChan)
				return
			}

		case <-tick:
			if buf.Len() > 0 {
				newMessage := &LogMessage{
					Filename: filename,
					Lines:    buf.Bytes(),
					Position: position,
				}

				batchChan <- newMessage
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
