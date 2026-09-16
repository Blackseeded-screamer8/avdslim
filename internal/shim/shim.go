package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kdbhalala/avdslim/internal/config"
)

const shimHeader = "# avdslim emulator shim"

func GetEmulatorBinaryPaths() (emuPath, realPath string, err error) {
	sdkDir := config.GetAndroidSdkDir()
	if sdkDir == "" {
		return "", "", fmt.Errorf("android SDK directory not found ($ANDROID_HOME or default SDK path)")
	}

	emuDir := filepath.Join(sdkDir, "emulator")
	binaryName := "emulator"
	realName := "emulator.real"
	if runtime.GOOS == "windows" {
		binaryName = "emulator.exe"
		realName = "emulator.real.exe"
	}

	emuPath = filepath.Join(emuDir, binaryName)
	realPath = filepath.Join(emuDir, realName)

	if _, err := os.Stat(emuDir); os.IsNotExist(err) {
		return "", "", fmt.Errorf("emulator directory %s does not exist", emuDir)
	}
	return emuPath, realPath, nil
}

func IsShimInstalled() (bool, error) {
	emuPath, realPath, err := GetEmulatorBinaryPaths()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(realPath); err == nil {
		data, err := os.ReadFile(emuPath)
		if err == nil && strings.Contains(string(data), shimHeader) {
			return true, nil
		}
	}
	return false, nil
}

func InstallShim(defaultRam int) error {
	emuPath, realPath, err := GetEmulatorBinaryPaths()
	if err != nil {
		return err
	}

	if defaultRam <= 0 {
		defaultRam = 1024
	}

	// Check if already installed
	alreadyInstalled, _ := IsShimInstalled()
	if alreadyInstalled {
		// Update existing shim with new RAM if changed
		return writeShimScript(emuPath, realPath, defaultRam)
	}

	// Verify original emulator exists
	if _, err := os.Stat(emuPath); os.IsNotExist(err) {
		return fmt.Errorf("emulator binary not found at %s", emuPath)
	}

	// Check if emuPath is already a script or real binary
	data, _ := os.ReadFile(emuPath)
	if strings.Contains(string(data), shimHeader) {
		return writeShimScript(emuPath, realPath, defaultRam)
	}

	// Move original emulator to emulator.real
	if err := os.Rename(emuPath, realPath); err != nil {
		return fmt.Errorf("failed to backup original emulator: %w", err)
	}

	// Write shim script
	if err := writeShimScript(emuPath, realPath, defaultRam); err != nil {
		// Rollback
		_ = os.Rename(realPath, emuPath)
		return fmt.Errorf("failed to install shim: %w", err)
	}

	return nil
}

func UninstallShim() error {
	emuPath, realPath, err := GetEmulatorBinaryPaths()
	if err != nil {
		return err
	}

	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		return fmt.Errorf("shim is not installed (original %s not found)", realPath)
	}

	// Remove shim file
	_ = os.Remove(emuPath)

	// Restore original binary
	if err := os.Rename(realPath, emuPath); err != nil {
		return fmt.Errorf("failed to restore original emulator: %w", err)
	}

	return nil
}

func writeShimScript(emuPath, realPath string, defaultRam int) error {
	var script string

	if runtime.GOOS == "windows" {
		script = fmt.Sprintf(`@echo off
rem %s — transparently injects low-memory flags
setlocal enabledelayedexpansion

set "REAL_EMU=%%~dp0emulator.real.exe"
if not exist "!REAL_EMU!" (
    echo [avdslim] Error: Real emulator binary not found at !REAL_EMU! 1>&2
    exit /b 1
)

set HAS_MEM=0
set HAS_LOWRAM=0
set NO_LOWRAM=0
set HAS_SNAP=0
set AVD_NAME=

set PREV=
for %%%%A in (%%*) do (
    if "%%%%~A"=="-memory" set HAS_MEM=1
    if "%%%%~A"=="-lowram" set HAS_LOWRAM=1
    if "%%%%~A"=="--no-lowram" set NO_LOWRAM=1
    if "%%%%~A"=="-no-lowram" set NO_LOWRAM=1
    if "%%%%~A"=="--no-slim" set NO_SLIM=1
    if "%%%%~A"=="-no-slim" set NO_SLIM=1
    if "%%%%~A"=="--headless" set HEADLESS=1
    if "%%%%~A"=="-no-window" set HEADLESS=1
    if "%%%%~A"=="-snapshot" set HAS_SNAP=1
    if "%%%%~A"=="-no-snapshot" set HAS_SNAP=1
    if "%%%%~A"=="--cold" set HAS_SNAP=1
    if "!PREV!"=="-avd" set "AVD_NAME=%%%%~A"
    set "PREV=%%%%~A"
)

set EXTRA=
if !NO_SLIM! EQU 0 (
    if !HAS_MEM! EQU 0 set EXTRA=!EXTRA! -memory %d
    if !HAS_LOWRAM! EQU 0 if !NO_LOWRAM! EQU 0 set EXTRA=!EXTRA! -lowram
    if !HEADLESS! EQU 1 set EXTRA=!EXTRA! -no-window
    if !HAS_SNAP! EQU 0 if defined AVD_NAME (
        if exist "%%USERPROFILE%%\.android\avd\!AVD_NAME!.avd\snapshots\avdslim_clean" (
            set EXTRA=!EXTRA! -snapshot avdslim_clean -no-snapshot-save
        )
    )
    set EXTRA=!EXTRA! -no-audio -camera-back none -camera-front none
)

"!REAL_EMU!" !EXTRA! %%*
`, shimHeader, defaultRam)
	} else {
		script = fmt.Sprintf(`#!/bin/bash
%s — transparently injects low-memory flags
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REAL_EMU="$DIR/emulator.real"

if [ ! -f "$REAL_EMU" ]; then
    echo "❌ [avdslim] Error: Real emulator binary not found at $REAL_EMU" >&2
    exit 1
fi

HAS_MEM=0
HAS_LOWRAM=0
NO_LOWRAM=0
NO_SLIM=0
HEADLESS=0
HAS_SNAP=0
AVD_NAME=""

PREV=""
for arg in "$@"; do
    if [ "$arg" = "-memory" ]; then HAS_MEM=1; fi
    if [ "$arg" = "-lowram" ]; then HAS_LOWRAM=1; fi
    if [ "$arg" = "--no-lowram" ] || [ "$arg" = "-no-lowram" ]; then NO_LOWRAM=1; fi
    if [ "$arg" = "--no-slim" ] || [ "$arg" = "-no-slim" ]; then NO_SLIM=1; fi
    if [ "$arg" = "--headless" ] || [ "$arg" = "-no-window" ]; then HEADLESS=1; fi
    if [ "$arg" = "-snapshot" ] || [ "$arg" = "-no-snapshot" ] || [ "$arg" = "--cold" ]; then HAS_SNAP=1; fi
    if [ "$PREV" = "-avd" ]; then AVD_NAME="$arg"; fi
    PREV="$arg"
done

EXTRA=()
if [ "$NO_SLIM" -ne 1 ]; then
    if [ "$HAS_MEM" -eq 0 ]; then
        EXTRA+=("-memory" "%d")
    fi
    if [ "$HAS_LOWRAM" -eq 0 ] && [ "$NO_LOWRAM" -eq 0 ]; then
        EXTRA+=("-lowram")
    fi
    if [ "$HEADLESS" -eq 1 ]; then
        EXTRA+=("-no-window")
    fi
    if [ "$HAS_SNAP" -eq 0 ] && [ -n "$AVD_NAME" ]; then
        SNAP_DIR="$HOME/.android/avd/${AVD_NAME}.avd/snapshots/avdslim_clean"
        if [ -d "$SNAP_DIR" ]; then
            EXTRA+=("-snapshot" "avdslim_clean" "-no-snapshot-save")
        fi
    fi
    EXTRA+=("-no-audio" "-camera-back" "none" "-camera-front" "none")
fi

exec "$REAL_EMU" "${EXTRA[@]}" "$@"
`, shimHeader, defaultRam)
	}

	if err := os.WriteFile(emuPath, []byte(script), 0755); err != nil {
		return err
	}
	return os.Chmod(emuPath, 0755)
}
