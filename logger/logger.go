package logger

import "log"

// Logger is an interface for a logger object
type Logger interface {
	Error(format string, params ...interface{})
	Warning(format string, params ...interface{})
	Note(format string, params ...interface{})
	Panic(format string, params ...interface{})
}

type defaultLogger struct{}

var currentLogger Logger = defaultLogger{}

// DefaultLogger returns a default logger (which calls log.Printf)
func DefaultLogger() Logger {
	return defaultLogger{}
}

// SetLogger sets the logger to be used by the package
func SetLogger(logger Logger) {
	currentLogger = logger
}

// Error prints an error message into the log
func Error(format string, params ...interface{}) {
	currentLogger.Error(format, params...)
}

// Warning prints a warning message into the log
func Warning(format string, params ...interface{}) {
	currentLogger.Warning(format, params...)
}

// Note prints a note into the log
func Note(format string, params ...interface{}) {
	currentLogger.Note(format, params...)
}

// Panic prints an error into the log and calls panic()
func Panic(format string, params ...interface{}) {
	currentLogger.Panic(format, params...)
}

// Error prints an error message into the log
func (d defaultLogger) Error(format string, params ...interface{}) {
	log.Printf("[Error] "+format, params...)
}

// Warning prints a warning message into the log
func (d defaultLogger) Warning(format string, params ...interface{}) {
	log.Printf("[Warning] "+format, params...)
}

// Note prints a note into the log
func (d defaultLogger) Note(format string, params ...interface{}) {
	log.Printf("[Note] "+format, params...)
}

// Panic prints an error into the log and calls panic()
func (d defaultLogger) Panic(format string, params ...interface{}) {
	log.Panicf("[Panic] "+format, params...)
}
