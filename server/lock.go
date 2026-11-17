package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"
)

// lockFile exists while quiz.exe runs, so a second instance refuses to start
// and cannot corrupt autosave.json (plan D20).
const lockFile = "quiz.lock"

// acquireLock creates the lock file exclusively. The returned func removes it
// on clean exit. A stale lock after a crash must be deleted by hand; the error
// message says so.
func acquireLock(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			old, _ := os.ReadFile(path)
			info := strings.TrimSpace(string(old))
			if info != "" {
				info = " (" + info + ")"
			}
			return nil, fmt.Errorf("quiz.lock exists%s: another quiz.exe is running. If not, delete quiz.lock and start again", info)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "pid=%d started=%s\n", os.Getpid(), time.Now().Format(time.RFC3339))
	if err := f.Close(); err != nil {
		return nil, err
	}
	return func() { os.Remove(path) }, nil
}
