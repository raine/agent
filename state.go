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
)

type GlobalState struct {
	sync.RWMutex

	Data *GlobalStateData
	File *os.File
}

type GlobalStateData struct {
	Version string            `json:"version"`
	States  map[string]*State `json:"states,omitempty"`
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

		gs.Data = &globalStateData
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

func (gs *GlobalState) deleteState(filename string) {
	delete(gs.Data.States, filename)
}

func (gs *GlobalState) getState(filename string) *State {
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

// DeleteState Removes state from GlobalState and writes changes to disk
func DeleteState(filename string) error {
	globalState.deleteState(filename)

	err := globalState.persistState()
	if err != nil {
		return err
	}

	return nil
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
		PersistState(filename, state.Checksum, state.Offset)
		os.Remove(LegacyStateFilename(filename))
		return state
	} else {
		state = globalState.getState(filename)
	}

	return state
}

//PersistState Creates a *State and records that in GlobalState.
func PersistState(filename string, checksum uint32, offset int64) error {
	globalState.saveState(filename, &State{Checksum: checksum, Offset: offset})

	err := globalState.persistState()
	if err != nil {
		return err
	}

	return nil
}

//StateFilename returns legacy StateFilename for a given filename.
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
