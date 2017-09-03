package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"

	"github.com/timberio/tail"
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

	var seekInfo *tail.SeekInfo
	start := &tail.SeekInfo{0, io.SeekStart}
	end := &tail.SeekInfo{0, io.SeekEnd}
	state, err := loadState(statefile)
	if err != nil {
		// TODO: not an error if state file didn't exist
		logger.Errorf("error determining start point: %s", err)
		seekInfo = end
	} else {
		checksum, err := calculateChecksum(filename)
		if err != nil {
			logger.Errorf("error checksumming: %s", err)
			seekInfo = end
		} else {
			if checksum == state.Checksum {
				// state file is applicable
				seekInfo = &tail.SeekInfo{state.Offset, io.SeekStart}
			} else {
				// file has been rotated
				seekInfo = start
			}
		}
	}

	inner, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Poll:     poll,
		Location: seekInfo,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal(err)
	}

	go func() {
		for {
			select {
			case line, ok := <-inner.Lines:
				if ok {
					if err := line.Err; err != nil {
						logger.Errorf("error reading from %s: %s", filename, err)
					} else {
						ch <- line.Text
					}
				} else {
					close(ch)
					logger.Infof("stopped tailing %s at offset %d", filename, inner.LastOffset)
					checksum, err := calculateChecksum(filename)
					if err == nil {
						if err := persistState(statefile, checksum, inner.LastOffset); err != nil {
							logger.Errorf("error persisting tail state: %s", err)
						}
					} else {
						logger.Errorf("error calculating checksum: %s", err)
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
	Checksum uint32
	Offset   int64
}

func loadState(statefile string) (*State, error) {
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

	return &state, nil
}

func persistState(statefile string, checksum uint32, offset int64) error {
	f, err := os.Create(statefile)
	if err != nil {
		return err
	}

	b, err := json.Marshal(State{Checksum: checksum, Offset: offset})
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

func calculateChecksum(file string) (uint32, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}

	b := make([]byte, 256)
	bytesRead, err := f.ReadAt(b, 0)
	if err != nil && err != io.EOF {
		return 0, err
	}

	// TODO: handle this more robustly
	if bytesRead < 256 {
		logger.Warnf("read %d bytes for checksum instead of 256", bytesRead)
	}

	return crc32.ChecksumIEEE(b), nil
}

func statefilePath(target string) string {
	return fmt.Sprintf("%s-state.json", path.Base(target))
}

type ReaderTailer struct {
	lines chan string
}

func NewReaderTailer(r io.Reader, quit chan bool) *ReaderTailer {
	ch := make(chan string)
	innerCh := make(chan string)
	scanner := bufio.NewScanner(r)

	go func() {
		for scanner.Scan() {
			innerCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			logger.Errorf("error reading stdin: ", err)
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

	return &ReaderTailer{lines: ch}
}

func (r *ReaderTailer) Lines() chan string {
	return r.lines
}
