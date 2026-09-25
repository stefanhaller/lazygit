package gocui

// Temporary: traces the color scheme detection to the file named by
// LAZYGIT_TTY_TRACE, for finding out why it doesn't work in Windows Terminal.

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	traceMutex sync.Mutex
	traceFile  *os.File
	traceStart = time.Now()
)

func init() {
	if path := os.Getenv("LAZYGIT_TTY_TRACE"); path != "" {
		traceFile, _ = os.Create(path)
	}
}

func trace(format string, args ...any) {
	if traceFile == nil {
		return
	}

	traceMutex.Lock()
	defer traceMutex.Unlock()

	elapsed := float64(time.Since(traceStart).Microseconds()) / 1000
	fmt.Fprintf(traceFile, "%9.1f ms  %s\n", elapsed, fmt.Sprintf(format, args...))
	_ = traceFile.Sync()
}

func traceBytes(p []byte) string {
	const limit = 160
	if len(p) > limit {
		return fmt.Sprintf("%q... (%d bytes)", p[:limit], len(p))
	}
	return fmt.Sprintf("%q", p)
}
