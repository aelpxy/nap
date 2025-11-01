package app

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type LockManager struct {
	lockDir string
}

var globalLockManager *LockManager

func GetGlobalLockManager() *LockManager {
	if globalLockManager == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		lockDir := filepath.Join(homeDir, ".nap", "locks")
		os.MkdirAll(lockDir, 0755)

		globalLockManager = &LockManager{
			lockDir: lockDir,
		}
	}
	return globalLockManager
}

func (lm *LockManager) TryLock(appName string, timeout time.Duration) error {
	lockFile := filepath.Join(lm.lockDir, appName+".lock")

	deadline := time.Now().Add(timeout)

	for {
		f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
			f.Close()
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("operation in progress, please wait")
		}

		data, err := os.ReadFile(lockFile)
		if err == nil {
			var pid int
			if n, _ := fmt.Sscanf(string(data), "%d", &pid); n == 1 {
				err := syscall.Kill(pid, 0)
				if err != nil {
					os.Remove(lockFile)
					continue
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (lm *LockManager) Unlock(appName string) {
	lockFile := filepath.Join(lm.lockDir, appName+".lock")
	os.Remove(lockFile)
}

func (lm *LockManager) IsLocked(appName string) bool {
	lockFile := filepath.Join(lm.lockDir, appName+".lock")
	_, err := os.Stat(lockFile)
	return err == nil
}
