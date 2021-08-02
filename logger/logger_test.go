package logger_test

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"ufygobot/logger"
)

// TestErrorLogging verifies that logger.Error outputs message i an expected format
func TestErrorLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	logger.Error("Sample error %d %d %d", 1, 2, 3)
	if !strings.HasSuffix(buf.String(), "[Error] Sample error 1 2 3\n") {
		t.Errorf("Unexpected logging for error: %s", buf.String())
	}
}

// TestWarningLogging verifies warning format
func TestWarningLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	logger.Warning("Sample warning %s", "1, 2, 3")
	if !strings.HasSuffix(buf.String(), "[Warning] Sample warning 1, 2, 3\n") {
		t.Errorf("Unexpected logging for warning: %s", buf.String())
	}
}

// TestNoteLogging verifies Note output
func TestNoteLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	logger.Note("Sample note %d", 999)
	if !strings.HasSuffix(buf.String(), "[Note] Sample note 999\n") {
		t.Errorf("Unexpected logging for note: %s", buf.String())
	}
}
