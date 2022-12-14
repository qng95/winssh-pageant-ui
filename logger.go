package main

import (
	"fmt"
	"log"
	"os"

	"path/filepath"
)

type LoggerType struct{}

var (
	Logger *LoggerType = &LoggerType{}
)

func (l *LoggerType) Init() {
	logFilePath, _ := os.OpenFile(filepath.Join(APP_LOGS_DIR, STARTUP_DATE+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	log.SetOutput(logFilePath)
}

func (l *LoggerType) Info(format string, v ...interface{}) {
	l.Log("INFO", format, v...)
}

func (l *LoggerType) Error(format string, v ...interface{}) {
	l.Log("ERROR", format, v...)
}

func (l *LoggerType) Panic(format string, v ...interface{}) {
	l.Log("PANIC", format, v...)
	panic(fmt.Sprintf(format, v...))
}

func (l *LoggerType) Fatal(exitcode int, format string, v ...interface{}) {
	l.Log("FATAL", format, v...)
	os.Exit(exitcode)
}

func (l *LoggerType) Log(level string, format string, v ...interface{}) {
	log.Printf(level + ": " + format + "\n", v...)
}

