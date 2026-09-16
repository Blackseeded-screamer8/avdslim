package config

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultsFilePath is the per-user file of default flags, e.g.
// ~/Library/Application Support/avdslim/defaults on macOS or
// ~/.config/avdslim/defaults on Linux. Empty if no config dir exists.
func DefaultsFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "avdslim", "defaults")
}

// LoadDefaults returns the flags in the defaults file (whitespace-separated,
// '#' starts a comment). A missing file yields no flags.
func LoadDefaults() []string {
	path := DefaultsFilePath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var flags []string
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		flags = append(flags, strings.Fields(line)...)
	}
	return flags
}
