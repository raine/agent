package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGlobalStateNew(test *testing.T) {
	globalState = NewGlobalState()

	if globalState.Data == nil {
		test.Fatalf("expected GlobalState Data to be initalized")
	}
}

func TestGlobalStateLoadWithFile(test *testing.T) {
	file, err := ioutil.TempFile("", "global-state-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	expected := &GlobalStateData{
		Version: "test",
		States: map[string]*State{
			"testfile": &State{
				Checksum: 12345,
				Offset:   100,
			},
		},
	}

	json, _ := json.Marshal(expected)
	file.WriteAt(json, 0)

	globalState = NewGlobalState()
	globalState.Load(file.Name())

	if !cmp.Equal(globalState.Data.States, expected.States) {
		test.Fatalf("Expected in memory global state to equal on disk global state")
	}
}

func TestGlobalStateLoadFileWithBadData(test *testing.T) {
	file, err := ioutil.TempFile("", "global-state-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	file.Write([]byte("Garbage"))

	globalState = NewGlobalState()
	globalState.Load(file.Name())

	if !cmp.Equal(globalState.Data.States, NewGlobalState().Data.States) {
		test.Fatal("Expected state to be initialized successfully")
	}
}

func TestGlobalStateLoadFileWithBadJSON(test *testing.T) {
	file, err := ioutil.TempFile("", "global-state-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	file.Write([]byte("{}"))

	globalState = NewGlobalState()
	globalState.Load(file.Name())

	if !cmp.Equal(globalState.Data.States, NewGlobalState().Data.States) {
		test.Fatal("Expected state to be initialized successfully")
	}
}

func TestGlobalStateLoadWithoutFile(test *testing.T) {
	filename := "tmp/global-state-test"
	defer func() {
		os.Remove(filename)
		os.Remove(filepath.Dir(filename))
	}()

	globalState = NewGlobalState()

	err := globalState.Load(filename)
	if err != nil {
		test.Fatalf("Expected load without file to succeed but got: %s", err)
	}

	if _, err := os.Stat(filename); err != nil {
		test.Fatalf("Expected %s to be created", filename)
	}

	err = globalState.Load(filename)
	if err != nil {
		test.Fatalf("Expected load with empty state written to succeed but got: %s", err)
	}
}

func TestDeleteState(test *testing.T) {
	file, err := ioutil.TempFile("", "global-state-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateData := &GlobalStateData{
		Version: "test",
		States: map[string]*State{
			"testfile": &State{
				Checksum: 12345,
				Offset:   100,
			},
		},
	}

	globalState = NewGlobalState()
	globalState.Data = globalStateData
	globalState.File = file

	DeleteState("testfile")

	if globalState.getState("testfile") != nil {
		test.Fatal("expected testfile state to be deleted")
	}

	globalState.Load(file.Name())

	if globalState.getState("testfile") != nil {
		test.Fatal("expected testfile state to be removed from disk")
	}
}

func TestLoadStateWithLegacyStatefile(test *testing.T) {
	filename := "tmp/testfile"
	globalStateFile := "tmp/global-statefile"
	stateFilename := LegacyStateFilename(filename)
	stateFile, err := os.Create(stateFilename)
	if err != nil {
		panic(err)
	}
	defer func() {
		os.Remove(stateFilename)
		os.Remove(globalStateFile)
		os.Remove(filepath.Dir(stateFilename))
	}()

	state := &State{
		Checksum: 12345,
		Offset:   100,
	}

	json, _ := json.Marshal(state)
	stateFile.WriteAt(json, 0)

	globalState = NewGlobalState()
	globalState.Load(globalStateFile)
	loadedState := LoadState(filename)

	storedState := globalState.getState(filename)
	if !cmp.Equal(storedState, loadedState) {
		test.Fatal("expected state and global state to be equal")
	}
	globalState.PersistState()

	globalState.Load(globalStateFile)
	storedState = globalState.getState(filename)
	if !cmp.Equal(storedState, loadedState) {
		test.Fatal("expected state to be read from disk")
	}
}

func TestLoadStateWithGlobalState(test *testing.T) {
	file, err := ioutil.TempFile("", "global-state-test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	globalStateData := &GlobalStateData{
		Version: "test",
		States: map[string]*State{
			"testfile": &State{
				Checksum: 12345,
				Offset:   100,
			},
		},
	}

	json, _ := json.Marshal(globalStateData)
	file.WriteAt(json, 0)
}

func TestLoadStateWithNoState(test *testing.T) {
	filename := "file-does-not-exist"

	globalState = NewGlobalState()
	state := LoadState(filename)

	if state != nil {
		test.Fatal("Expected state to be nil")
	}
}

func TestPersistState(test *testing.T) {
	filename := "file-state-to-persist"
	var checksum uint32 = 12345
	var offset int64 = 100

	globalStateFile := "tmp/global-statefile"
	defer func() {
		os.Remove(globalStateFile)
		os.Remove(filepath.Dir(globalStateFile))
	}()

	globalState = NewGlobalState()
	globalState.Load(globalStateFile)

	UpdateState(filename, checksum, offset)
	globalState.PersistState()

	storedState := globalState.getState(filename)
	if storedState.Checksum != checksum || storedState.Offset != offset {
		test.Fatal("expected state to be written to global state data")
	}

	globalState.Load(globalStateFile)

	storedState = globalState.getState(filename)
	if storedState.Checksum != checksum || storedState.Offset != offset {
		test.Fatal("expected state to be written to global state data")
	}
}

func TestStateFilename(test *testing.T) {
	filename := "/var/log/testfile.log"
	expected := "testfile.log-state.json"

	stateFilename := LegacyStateFilename(filename)
	if expected != stateFilename {
		test.Fatalf("expected %s, got %s", expected, stateFilename)
	}
}
