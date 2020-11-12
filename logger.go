package stanclient

import (
	"fmt"
	"os"
)

// ClientLogger implements clients logging needs
type ClientLogger interface {
	Info(args ...interface{})
	Fatal(args ...interface{})
}

// EmptyLogger implements ClientLogger
type EmptyLogger struct{}

// Info does nothing
func (l *EmptyLogger) Info(args ...interface{}) {}

// Fatal only exists
func (l *EmptyLogger) Fatal(args ...interface{}) {
	os.Exit(1)
}

// FmtLogger outputs info on stdout
type FmtLogger struct{}

// Info printf %v
func (l *FmtLogger) Info(args ...interface{}) {
	format := ""
	for range args {
		format += "%v "
	}
	fmt.Printf(format+"\n", args...)
}

// Fatal printf %v then exits
func (l *FmtLogger) Fatal(args ...interface{}) {
	fmt.Printf("%v\n", args...)
	os.Exit(1)
}
