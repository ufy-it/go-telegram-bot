package state

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type StateIO interface {
	Load() ([]byte, error)
	Save([]byte) error
}

type fileStateIO struct {
	filename string
}

func NewFileState(filename string) StateIO {
	return fileStateIO{filename: filename}
}

func (f fileStateIO) Load() ([]byte, error) {
	file, err := ioutil.ReadFile(f.filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read state file %s: %v", f.filename, err)
	}
	return file, nil
}

func (f fileStateIO) Save(file []byte) error {
	if f.filename == "" {
		return errors.New("cannot save state: filename is not set")
	}
	tempFile, err := ioutil.TempFile(".", "persistant")
	if err != nil {
		return fmt.Errorf("cannot ctreate temp file: %v", err)
	}
	_, err = tempFile.Write(file)
	if err != nil {
		return fmt.Errorf("cannot write state to a temp file: %v", err)
	}
	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("cannot write state to a temp file: %v", err)
	}

	err = os.Rename(tempFile.Name(), f.filename)
	if err != nil {
		return fmt.Errorf("cannot rename temp file to the target file %s: %v", f.filename, err)
	}
	return nil
}
