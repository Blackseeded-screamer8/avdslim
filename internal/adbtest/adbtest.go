// Package adbtest provides a fake adb for tests that need no emulator.
package adbtest

import (
	"os"
	"path/filepath"
)

// script emulates the adb calls avdslim makes, keeping all device state as
// files next to itself:
//
//	devices             `adb devices` output
//	avd_<serial>        `emu avd name` output
//	prop_<name>         getprop values
//	packages            installed packages, one per line
//	<ns>_<name>         settings values
//	state               the guest state JSON
//	calls               every invocation, one per line (appended)
//
// Failure switches (create the file to trigger):
//
//	pm_broken           `pm list packages` fails
//	enable_fails        packages (one per line) that `pm enable` rejects
//	readonly            writing the state file fails
const script = `#!/bin/bash
D="$(dirname "$0")"
echo "$*" >> "$D/calls"
case "$1" in
  devices) cat "$D/devices" 2>/dev/null || printf "List of devices attached\n"; exit 0 ;;
  version) echo "Android Debug Bridge version 1.0.41 (fake)"; exit 0 ;;
esac
if [ "$1" = "-s" ] && [ "$3" = "emu" ]; then
  cat "$D/avd_$2" 2>/dev/null || echo "Test_AVD"
  exit 0
fi
[ "$3" = "shell" ] || exit 0
shift 3 # drop: -s SERIAL shell
case "$1 $2" in
  "settings get") cat "$D/$3_$4" 2>/dev/null || echo null ;;
  "settings put") printf '%s' "$5" > "$D/$3_$4" ;;
  "settings delete") rm -f "$D/$3_$4" ;;
  "pm list")
    [ -f "$D/pm_broken" ] && { echo "cmd: Can't find service: package"; exit 1; }
    if [ -f "$D/packages" ]; then sed 's/^/package:/' "$D/packages"; else echo "package:android"; fi ;;
  "pm enable")
    grep -qx "$3" "$D/enable_fails" 2>/dev/null && { echo "Error: Unknown package: $3"; exit 1; }
    echo "Package $3 new state: enabled" ;;
  "pm disable-user") echo "Package $5 new state: disabled-user" ;;
  "getprop "*) cat "$D/prop_$2" 2>/dev/null ;;
  "cat "*) cat "$D/state" 2>/dev/null ;;
  "ls "*) [ -f "$D/state" ] && echo "$2" ;;
  "rm -f") rm -f "$D/state" ;;
  "echo "*)
    [ -f "$D/readonly" ] && { echo "/system/bin/sh: can't create $4: Read-only file system"; exit 1; }
    v="$2"; v="${v#\'}"; printf '%s' "${v%\'}" > "$D/state" ;;
esac
exit 0
`

// Install writes the fake adb into dir and returns its path.
func Install(dir string) (string, error) {
	p := filepath.Join(dir, "adb")
	return p, os.WriteFile(p, []byte(script), 0755)
}
