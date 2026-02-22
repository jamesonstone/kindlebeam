package app

import (
	"fmt"
	"os"
)

type Logger struct {
	verbose bool
}

func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

func (l *Logger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ℹ️ "+format+"\n", args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "🔍 "+format+"\n", args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
}

func (l *Logger) Successf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "✅ "+format+"\n", args...)
}
