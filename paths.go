package main

import (
	"os"
	"path/filepath"
)

// userHomePath builds a path under the current user's home directory so source can be shared safely.
func userHomePath(parts ...string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(parts...)
	}
	return filepath.Join(append([]string{home}, parts...)...)
}
