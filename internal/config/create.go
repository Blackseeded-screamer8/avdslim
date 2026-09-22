package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// HostAbi is the system-image ABI that runs natively on this machine.
func HostAbi() string {
	if runtime.GOARCH == "arm64" {
		return "arm64-v8a"
	}
	return "x86_64"
}

// SlimImages lists installed system images avdslim recommends, newest API
// first, as sdkmanager package ids. Only the plain "google_apis" tag qualifies:
// that excludes Play Store images (no adb root, updater churn) and 16 KB
// page-size images (4 GB RAM floor). api > 0 keeps only that major API level.
func SlimImages(sdkDir string, api int) []string {
	type image struct {
		level float64
		pkg   string
	}
	var found []image
	dirs, _ := filepath.Glob(filepath.Join(sdkDir, "system-images", "android-*", "google_apis", HostAbi()))
	for _, d := range dirs {
		platform := filepath.Base(filepath.Dir(filepath.Dir(d)))
		level, err := strconv.ParseFloat(strings.TrimPrefix(platform, "android-"), 64)
		if err != nil || (api > 0 && int(level) != api) {
			continue // ponytail: codename previews (android-Baklava) are skipped
		}
		found = append(found, image{level, "system-images;" + platform + ";google_apis;" + HostAbi()})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].level > found[j].level })
	pkgs := make([]string, len(found))
	for i, f := range found {
		pkgs[i] = f.pkg
	}
	return pkgs
}

// FindAvdmanager locates avdmanager: PATH, then the SDK's cmdline-tools
// (latest/ or the highest versioned dir, e.g. 23.0/). "" when not installed.
func FindAvdmanager() string {
	name := "avdmanager"
	if runtime.GOOS == "windows" {
		name = "avdmanager.bat"
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	sdk := GetAndroidSdkDir()
	if sdk == "" {
		return ""
	}
	candidates := []string{filepath.Join(sdk, "cmdline-tools", "latest", "bin", name)}
	versioned, _ := filepath.Glob(filepath.Join(sdk, "cmdline-tools", "*", "bin", name))
	sort.Sort(sort.Reverse(sort.StringSlice(versioned))) // ponytail: lexical, fine until cmdline-tools 100.0
	candidates = append(candidates, versioned...)
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
