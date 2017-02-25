package mqttinflux

import (
    "log"
)

// Log Levels -----------------------------------------------------------------

func LogError(message string, v ...interface{}) {
    logLevel("ERROR", message, v...)
}

func LogWarning(message string, v ...interface{}) {
    logLevel("WARNING", message, v...)
}

func LogInfo(message string, v ...interface{}) {
    logLevel("INFO", message, v...)
}

func logLevel(level, message string, v ...interface{}) {
    m := level + " " + message
    log.Printf(m, v...)
}
