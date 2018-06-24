package mqttinflux

import (
	"log"
)

// Log Levels -----------------------------------------------------------------

// LogError emits a log message with level ERROR.
func LogError(message string, v ...interface{}) {
	logLevel("ERROR", message, v...)
}

// LogWarning emits a log message with level WARNING.
func LogWarning(message string, v ...interface{}) {
	logLevel("WARNING", message, v...)
}

// LogInfo emits a log message with level INFO.
func LogInfo(message string, v ...interface{}) {
	logLevel("INFO", message, v...)
}

func logLevel(level, message string, v ...interface{}) {
	m := level + " " + message
	log.Printf(m, v...)
}
