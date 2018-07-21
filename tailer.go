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

func NewFileTailer(filename string, poll bool, quit chan bool, stop chan bool) *FileTailer {
	logger.Infof("Creating new file tailer for %s", filename)

	ch := make(chan *LogMessage)

	var seekInfo *tail.SeekInfo
	start := &tail.SeekInfo{0, io.SeekStart}
	end := &tail.SeekInfo{0, io.SeekEnd}

	// Attempt to resume tailing
	state := LoadState(filename)
	if state == nil {
		logger.Warnf("Could not load state for file %s, agent will recognize new data only", filename)
		seekInfo = end

		var offset int64
		stat, err := os.Stat(filename)
		if err != nil {
			logger.Errorf("Failed to stat file %s: %s", filename, err)
			offset = 0
		} else {
			offset = stat.Size()
		}

		checksum, err := calculateChecksum(filename)
		if err != nil {
			logger.Errorf("Failed to checksum file %s: %s", filename, err)
		}

		// Record state of file in globalState
		UpdateState(filename, checksum, offset)
	} else {
		checksum, err := calculateChecksum(filename)
		if err != nil {
			logger.Errorf("Failed to generate checksum for file %s, agent will recognize new data only: %s", filename, err)
			seekInfo = end
		} else {
			if checksum == state.Checksum {
				// state file is applicable
				seekInfo = &tail.SeekInfo{state.Offset, io.SeekStart}
				logger.Infof("Checksum for %s matched recorded state, resuming - offset: %d, seekstart: %d", filename, state.Offset, io.SeekStart)
			} else {
				// file has been rotated
				logger.Infof("Checksum for %s does not match recorded state. Reading from beginning of file.", filename)
				seekInfo = start
			}
		}
	}

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
						position, _ := inner.Tell()

						ch <- &LogMessage{
							Filename: filename,
							Lines:    []byte(line.Text),
							Position: position,
						}
					}
				} else {
					close(ch)
					logger.Infof("Stopped tailing %s at offset %d", filename, inner.LastOffset)
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
