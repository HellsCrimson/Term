package main

import "log"

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

var (
	loggingLevel = LevelDebug
)

type LoggingService struct{}

func (g *LoggingService) Log(message string, level string) {
	levels := map[string]int{
		LevelInfo:  1,
		LevelDebug: 2,
		LevelWarn:  3,
		LevelError: 4,
	}

	if _, ok := levels[level]; !ok {
		return // Invalid level
	}

	if levels[level] >= levels[loggingLevel] {
		log.Printf("[%s] %s", level, message)
	}
}
