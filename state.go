package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//LogMessage An interface to represent a LogMessage, which is one or more log lines to be sent to a centralized logging
// backend. This interface contains all the fields required for storing state within our GlobalState. For now, since
// we are only storing the state of tracked files, LogMessageFile is our only implementation, with other LogMessage
// structs containing placeholder/dummy methods.
// type LogMessage interface {
// 	Lines() []byte
// 	Position() int64
// 	Filename() string
// }

//LogMessageFile LogMessage for a log line from a file
type LogMessage struct {
	Filename string
	Lines    []byte
	Position int64
}

type GlobalState struct {
	sync.Mutex

	Data *GlobalStateData
	File *os.File

	flushTimer *time.Ticker
}

type GlobalStateData struct {
	sync.RWMutex

	Version string            `json:"version"`
	States  map[string]*State `json:"states"`
}

type State struct {
	Checksum uint32
	Offset   int64
}

// Provides global state to this package
var globalState = NewGlobalState()

//NewGlobalState return an empty *GlobalState with reference types allocated
func NewGlobalState() *GlobalState {
	globalStateData := &GlobalStateData{
		Version: version,
		States:  make(map[string]*State),
	}

	return &GlobalState{
		Data: globalStateData,
	}
}

//DefaultGlobalStateFilename returns a path for recording our global state. This path follows best practices for
// the supported operating systems.
func DefaultGlobalStateFilename() string {
	// For BSD based systems, the default path is under /var/db
	// https://www.freebsd.org/cgi/man.cgi?query=hier&apropos=0&sektion=7&manpath=FreeBSD+11.0-RELEASE+and+Ports&arch=default&format=html
	// http://netbsd.gw.com/cgi-bin/man-cgi?hier+7+NetBSD-current
	// https://man.openbsd.org/hier.7
	if strings.Contains(runtime.GOOS, "bsd") || runtime.GOOS == "darwin" {
		return "/var/db/timber-agent/statefile.json"
	} else {
		// On Linux based systems, the default path is under /var/lib
		return "/var/lib/timber-agent/statefile.json"
	}
}

//Load Reads GlobalState Data from stateFilename, or creates stateFilename if not found. Returns an error if we fail
// to create, read, or write to stateFile, as it is crucial to our execution.
func (gs *GlobalState) Load(stateFilename string) error {
	var stateFile *os.File

	// If we find a file, read from it
	if _, err := os.Stat(stateFilename); err == nil {
		stateFile, err = os.OpenFile(stateFilename, os.O_RDWR, os.ModePerm)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to open global statefile %s: %s", stateFilename, err))
		}

		bytes, err := ioutil.ReadAll(stateFile)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to read global statefile %s: %s", stateFilename, err))
		}

		var globalStateData GlobalStateData
		if err := json.Unmarshal(bytes, &globalStateData); err != nil {
			return errors.New(fmt.Sprintf("Unable to parse json from global statefile %s: %s", stateFilename, err))
		}

		// We could have malformed or empty fields in our statefile while
		// still being valid json. In this case, we need to ensure our
		// globalState is valid and does not contain incorrect values.
		// We accomplish this by preserving our default initial values,
		// which will be overwritten on next state update.
		//
		// We may ignore logs line here if configuraiton is not set to read the
		// file from beginning on discovery, but this will allow the agent
		// to boot and fix that statefile. Additionally, not setting read from
		// beginning already implies some data may be ignored.
		//
		// When unmarshalling JSON, objects not found will be set to the
		// default value for that type. For strings it is the empty string,
		// and for maps that is nil.
		if globalStateData.Version != "" {
			gs.Data.Version = globalStateData.Version
		}

		if globalStateData.States != nil {
			gs.Data.States = globalStateData.States
		}
	} else {
		// If we didn't find a file, we create the file and persist its empty state to disk
		stateFile, err = createGlobalStateFile(stateFilename)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to create global statefile %s: %s", stateFilename, err))
		}

		err = writeGlobalStateFile(stateFile, gs.Data)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to write global statefile %s: %s", stateFilename, err))
		}
	}

	// Save reference to file in globalState
	gs.File = stateFile

	return nil
}

//PersistState Write GlobalState to disk
func (gs *GlobalState) PersistState() error {
	return gs.persistState()
}

//Start Create and start ticker for writing GlobalState to disk on a timer
func (gs *GlobalState) Start() {
	gs.flushTimer = time.NewTicker(1 * time.Second)

	for _ = range gs.flushTimer.C {
		err := gs.persistState()
		if err != nil {
			logger.Fatal(err)
		}
	}
}

//Start Stop GlobalState ticker
func (gs *GlobalState) Stop() {
	gs.flushTimer.Stop()
}

func (gs *GlobalState) deleteState(filename string) {
	gs.Data.Lock()
	defer gs.Data.Unlock()

	delete(gs.Data.States, filename)
}

func (gs *GlobalState) getState(filename string) *State {
	gs.Data.RLock()
	defer gs.Data.RUnlock()

	return gs.Data.States[filename]
}

func (gs *GlobalState) persistState() error {
	gs.Lock()
	defer gs.Unlock()

	err := writeGlobalStateFile(gs.File, gs.Data)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to write to global statefile %s: %s", gs.File.Name(), err))
	}

	return nil
}

func (gs *GlobalState) saveState(filename string, state *State) {
	gs.Data.Lock()
	defer gs.Data.Unlock()

	gs.Data.States[filename] = state
}

func createGlobalStateFile(filename string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return nil, err
	}

	stateFile, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return stateFile, nil
}

func writeGlobalStateFile(file *os.File, data *GlobalStateData) error {
	json, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := file.Truncate(0); err != nil {
		return err
	}

	if _, err := file.WriteAt(json, 0); err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}

	return nil
}

//DeleteState Removes state from GlobalState
func DeleteState(filename string) {
	globalState.deleteState(filename)
}

//LoadState Returns initialized *State with data for given filename. We first look at our legacy statefile before
// our global state, so we can cleanup if needed.
func LoadState(filename string) *State {
	var state *State

	state, err := loadStateFile(filename)
	if err != nil {
		logger.Debugf("Unable to load statefile for %s: %s", filename, err)
	}

	if state != nil {
		UpdateState(filename, state.Checksum, state.Offset)
		os.Remove(LegacyStateFilename(filename))
		return state
	} else {
		state = globalState.getState(filename)
	}

	return state
}

//UpdateState Update globalState entry for filename in memory. globalState is flushed to disk on an interval and
// at agent shutdown.
func UpdateState(filename string, checksum uint32, offset int64) {
	globalState.saveState(filename, &State{Checksum: checksum, Offset: offset})
}

//UpdateStateChecksum Update globalState checksum for filename in memory
func UpdateStateChecksum(filename string, checksum uint32) error {
	state := globalState.getState(filename)
	if state == nil {
		return errors.New(fmt.Sprintf("Unable to read state for file %s", filename))
	}

	state.Checksum = checksum
	globalState.saveState(filename, state)

	return nil
}

//UpdateStateOffset Update globalState offset for filename in memory
func UpdateStateOffset(filename string, offset int64) error {
	state := globalState.getState(filename)
	if state == nil {
		return errors.New(fmt.Sprintf("Unable to read state for file %s", filename))
	}

	state.Offset = offset
	globalState.saveState(filename, state)

	return nil
}

//LegacyStateFilename returns legacy StateFilename for a given filename.
func LegacyStateFilename(filename string) string {
	return fmt.Sprintf("%s-state.json", path.Base(filename))
}

func loadStateFile(filename string) (*State, error) {
	f, err := os.Open(LegacyStateFilename(filename))
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
