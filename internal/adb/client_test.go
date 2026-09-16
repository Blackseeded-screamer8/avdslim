package adb

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeAdb emulates `adb -s X shell ...` for settings, the state file, and rm,
// storing each setting as a file in dir.
const fakeAdb = `#!/bin/bash
D="$(dirname "$0")"
shift 2; shift # drop: -s SERIAL shell
case "$1 $2" in
  "settings get") cat "$D/$3_$4" 2>/dev/null || echo null ;;
  "settings put") printf '%s' "$5" > "$D/$3_$4" ;;
  "settings delete") rm -f "$D/$3_$4" ;;
  "cat "*) cat "$D/state" 2>/dev/null ;;
  "rm -f") rm -f "$D/state" ;;
  "echo "*) v="$2"; v="${v#\'}"; printf '%s' "${v%\'}" > "$D/state" ;;
esac
`

func newFake(t *testing.T) (*Client, string) {
	if runtime.GOOS == "windows" {
		t.Skip("fake adb is a bash script")
	}
	dir := t.TempDir()
	adb := filepath.Join(dir, "adb")
	if err := os.WriteFile(adb, []byte(fakeAdb), 0755); err != nil {
		t.Fatal(err)
	}
	return &Client{adbPath: adb}, dir
}

func setting(t *testing.T, dir, ns, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, ns+"_"+name))
	if err != nil {
		return "<unset>"
	}
	return string(b)
}

func TestSlimRestoreIsExactInverse(t *testing.T) {
	c, dir := newFake(t)
	// User's own values: half-speed animations; background_process_limit unset.
	c.Exec("-s", "e", "shell", "settings", "put", "global", "window_animation_scale", "0.5")
	c.Exec("-s", "e", "shell", "settings", "put", "global", "auto_sync", "1")

	c.Slim("e", false, nil, map[string]bool{"sync": true})
	if got := setting(t, dir, "global", "window_animation_scale"); got != "0" {
		t.Fatalf("after slim animations = %q, want 0", got)
	}
	if got := setting(t, dir, "global", "background_process_limit"); got != "4" {
		t.Fatalf("after slim bglimit = %q, want 4", got)
	}
	if got := setting(t, dir, "global", "auto_sync"); got != "1" {
		t.Fatalf("skipped sync changed to %q", got)
	}

	// Re-slim without --skip: keeps the first originals (not the slimmed values)
	// and records sync's current value before changing it.
	c.Slim("e", false, nil, nil)
	c.Restore("e")

	for _, tc := range []struct{ ns, name, want string }{
		{"global", "window_animation_scale", "0.5"},
		{"global", "background_process_limit", "<unset>"},
		{"global", "auto_sync", "1"},
		{"secure", "location_mode", "<unset>"},
	} {
		if got := setting(t, dir, tc.ns, tc.name); got != tc.want {
			t.Errorf("after restore %s = %q, want %q", tc.name, got, tc.want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "state")); err == nil {
		t.Error("state file not removed")
	}
}
