package logger

import "log"

// Error prints an error message into the log
func Error(format string, params ...interface{}) {
	log.Printf("[Error] "+format, params...)
}

// Warning prints a warning message into the log
func Warning(format string, params ...interface{}) {
	log.Printf("[Warning] "+format, params...)
}

// Note prints a note into the log
func Note(format string, params ...interface{}) {
	log.Printf("[Note] "+format, params...)
}

// Panic prints an error into the log and calls panic()
func Panic(format string, params ...interface{}) {
	log.Printf("[Panic] "+format, params...)
}
