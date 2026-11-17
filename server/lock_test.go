package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quiz.lock")
	release, err := acquireLock(path)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
	if _, err := acquireLock(path); err == nil {
		t.Fatal("second lock accepted")
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock file not removed: %v", err)
	}
	release2, err := acquireLock(path)
	if err != nil {
		t.Fatalf("relock: %v", err)
	}
	release2()
}
