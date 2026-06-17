package logutil

import (
	"log"
	"os"
	"sync"
)

var (
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	once        sync.Once
)

// this to iniatialise the loggers with consistent flags and prefix
func Init(role string) {
	once.Do(func() {
		flags := log.LstdFlags | log.Lmicroseconds | log.Lshortfile
		prefix := "[" + role + "]"

		infoLogger = log.New(os.Stdout, prefix+"INFO: ", flags)
		warnLogger = log.New(os.Stderr, prefix+"WARN: ", flags)
		errorLogger = log.New(os.Stderr, prefix+"ERROR: ", flags)
		debugLogger = log.New(os.Stdout, prefix+"DEBUG: ", flags)
	})
}

func Info(format string, v ...any) {
	infoLogger.Printf(format, v...)
}

func Warn(format string, v ...any) {
	warnLogger.Printf(format, v...)
}

func Error(format string, v ...any) {
	errorLogger.Printf(format, v...)
}

func Debug(format string, v ...any) {
	debugLogger.Printf(format, v...)
}

func Fatal(format string, v ...any) {
	errorLogger.Fatalf(format, v...)
}
