package main

import (
	"bufio"
	"io"
	"log"

	"github.com/influxdata/tail"
)

type Tailer interface {
	Lines() chan string
}

type FileTailer struct {
	inner *tail.Tail
	lines chan string
}

func NewFileTailer(filename string, poll bool, quit chan bool) FileTailer {
	ch := make(chan string)
	// TODO: check for statefile with progress, seek there
	inner, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Poll:     poll,
		Location: &tail.SeekInfo{0, io.SeekEnd},
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case line, ok := <-inner.Lines:
				if ok {
					if err := line.Err; err != nil {
						log.Println("error reading from %s: %s", filename, err)
					} else {
						ch <- line.Text
					}
				} else {
					close(ch)
					// TODO: save progress
					return
				}

			case <-quit:
				inner.Stop()
			}
		}
	}()

	return FileTailer{inner: inner, lines: ch}
}

func (f *FileTailer) Lines() chan string {
	return f.lines
}

type ReaderTailer struct {
	lines chan string
}

func NewReaderTailer(r io.Reader, quit chan bool) ReaderTailer {
	ch := make(chan string)
	innerCh := make(chan string)
	scanner := bufio.NewScanner(r)

	go func() {
		for scanner.Scan() {
			innerCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Println("error reading stdin: ", err)
		}
		close(innerCh)
	}()

	go func() {
		for {
			select {
			case line, ok := <-innerCh:
				if !ok {
					close(ch)
					return
				}
				ch <- line
			case <-quit:
				close(ch)
				return
			}
		}
	}()

	return ReaderTailer{lines: ch}
}

func (r *ReaderTailer) Lines() chan string {
	return r.lines
}
