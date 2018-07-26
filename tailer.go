package main

import (
	"bufio"
	"hash/crc32"
	"io"
	"os"

	"github.com/timberio/tail"
)

type Tailer interface {
	Lines() chan *LogMessage
}

type FileTailer struct {
	filename string
	inner    *tail.Tail
	lines    chan *LogMessage
}

func NewFileTailer(filename string, readNewFileFromStart bool, poll bool, quit chan bool, stop chan bool) *FileTailer {
	logger.Infof("Creating new file tailer for %s", filename)

	ch := make(chan *LogMessage)

	var seekInfo *tail.SeekInfo
	start := &tail.SeekInfo{0, io.SeekStart}
	end := &tail.SeekInfo{0, io.SeekEnd}

	var newState *State

	state := LoadState(filename)
	if state != nil {
		// We treat a failed checksum the same as a non-matching checksum which results in reading the file from the
		// beginning. While we may send duplicate data, we prefer that over not sending new data.
		checksum, err := calculateChecksum(filename)
		if err != nil {
			logger.Errorf("Failed to checksum file %s: %s", filename, err)
		}

		if checksum == state.Checksum {
			logger.Infof("Checksum for %s matched recorded state, resuming - offset: %d, seekstart: %d", filename, state.Offset, io.SeekStart)
			seekInfo = &tail.SeekInfo{state.Offset, io.SeekStart}
		} else {
			logger.Infof("Checksum for %s does not match recorded state. Reading from beginning of file.", filename)
			seekInfo = start
			state.Offset = 0
		}

		newState = state
	} else {
		var msg string
		newState = &State{}

		if readNewFileFromStart {
			msg = "read from start of file"
			seekInfo = start
		} else {
			msg = "recognize new data only"
			seekInfo = end

			stat, err := os.Stat(filename)
			if err != nil {
				logger.Errorf("Failed to stat file %s: %s", filename, err)
			} else {
				newState.Offset = stat.Size()
			}
		}

		checksum, err := calculateChecksum(filename)
		if err != nil {
			logger.Errorf("Failed to checksum file %s: %s", filename, err)
		}
		newState.Checksum = checksum

		logger.Infof("New file detected %s, agent will %s", filename, msg)
	}

	// Write state of file to globalState, which may be redundant but handles all cases
	UpdateState(filename, newState.Checksum, newState.Offset)

	inner, err := tail.TailFile(filename, tail.Config{
		Follow:    true,
		ReOpen:    true,
		Poll:      poll,
		Location:  seekInfo,
		Logger:    logger,
		MustExist: true,
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
						logger.Errorf("Error reading from %s: %s", filename, err)
					} else {
						position := inner.Offset

						ch <- &LogMessage{
							Filename: filename,
							Lines:    []byte(line.Text),
							Position: position,
						}
					}
				} else {
					close(ch)
					checksum, err := calculateChecksum(filename)
					if err == nil {
						UpdateStateChecksum(filename, checksum)
					} else {
						logger.Errorf("Error calculating checksum: %s", err)
					}
					return
				}

			case <-quit:
				// looks like we're getting into here multiple times for some files?
				inner.Stop()

			case <-stop:
				inner.Stop()
			}

		}
	}()

	return &FileTailer{inner: inner, lines: ch, filename: filename}
}

func (f *FileTailer) Lines() chan *LogMessage {
	return f.lines
}

func (f *FileTailer) Wait() {
	f.inner.Wait()
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
		logger.Warnf("Read %d bytes for checksum instead of 256", bytesRead)
	}

	return crc32.ChecksumIEEE(b), nil
}

type ReaderTailer struct {
	lines chan *LogMessage
}

func NewReaderTailer(r io.Reader, quit chan bool) *ReaderTailer {
	logger.Info("Creating reader tailer")

	ch := make(chan *LogMessage)
	innerCh := make(chan string)
	scanner := bufio.NewScanner(r)

	go func() {
		for scanner.Scan() {
			innerCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			logger.Errorf("Error reading stdin: ", err)
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
				ch <- &LogMessage{
					Filename: "stdin",
					Lines:    []byte(line),
					Position: 0,
				}
			case <-quit:
				close(ch)
				return
			}
		}
	}()

	return &ReaderTailer{lines: ch}
}

func (r *ReaderTailer) Lines() chan *LogMessage {
	return r.lines
}
