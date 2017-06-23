package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/influxdata/tail"
)

type Tailer interface {
	Lines() chan string
}

type FileTailer struct {
	inner     *tail.Tail
	lines     chan string
	statefile string
}

func NewFileTailer(filename string, poll bool, quit chan bool) *FileTailer {
	ch := make(chan string)
	statefile := statefilePath(filename)

	end := &tail.SeekInfo{0, io.SeekEnd}
	seekInfo, err := findStartingPoint(statefile)
	if err != nil {
		// TODO: not an error if state file didn't exist
		log.Printf("error determining start point: %s", err)
		seekInfo = end
	}

	inner, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Poll:     poll,
		Location: seekInfo,
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
					log.Printf("stopped tailing %s at offset %d", filename, inner.LastOffset)
					if err := persistState(statefile, inner.LastOffset); err != nil {
						log.Printf("error persisting tail state: %s", err)
					}
					return
				}

			case <-quit:
				// looks like we're getting into here multiple times for some files?
				inner.Stop()
			}
		}
	}()

	return &FileTailer{inner: inner, lines: ch, statefile: statefile}
}

func (f *FileTailer) Lines() chan string {
	return f.lines
}

func (f *FileTailer) Wait() {
	f.inner.Wait()
}

func (f *FileTailer) RemoveStatefile() {
	os.Remove(f.statefile)
}

type State struct {
	Offset int64
}

func findStartingPoint(statefile string) (*tail.SeekInfo, error) {
	f, err := os.Open(statefile)
	if err != nil {
		return nil, err
	}

	b := make([]byte, 2048)
	bytesRead, err := f.Read(b)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(b[:bytesRead], &state); err != nil {
		return nil, err
	}

	return &tail.SeekInfo{state.Offset, io.SeekStart}, nil
}

func persistState(statefile string, offset int64) error {
	f, err := os.Create(statefile)
	if err != nil {
		return err
	}

	b, err := json.Marshal(State{Offset: offset})
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}

	if _, err := f.WriteAt(b, 0); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func statefilePath(target string) string {
	return fmt.Sprintf("%s-state.json", path.Base(target))
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
