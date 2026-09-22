package adb

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kdbhalala/avdslim/internal/adbtest"
)

func newFake(t *testing.T) (*Client, string) {
	if runtime.GOOS == "windows" {
		t.Skip("fake adb is a bash script")
	}
	dir := t.TempDir()
	adb, err := adbtest.Install(dir)
	if err != nil {
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
		{"global", "bluetooth_on", "<unset>"},
	} {
		if got := setting(t, dir, tc.ns, tc.name); got != tc.want {
			t.Errorf("after restore %s = %q, want %q", tc.name, got, tc.want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "state")); err == nil {
		t.Error("state file not removed")
	}
}

func TestResolveDevice(t *testing.T) {
	c, dir := newFake(t)

	// Case 1: No devices running
	if _, err := c.ResolveDevice(nil); err == nil {
		t.Fatal("expected error when no devices are running")
	}

	// Case 2: One device running (emulator-5554, AVD: Slim_Pixel_5)
	oneDevice := "List of devices attached\nemulator-5554\tdevice\n"
	if err := os.WriteFile(filepath.Join(dir, "devices"), []byte(oneDevice), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "avd_emulator-5554"), []byte("Slim_Pixel_5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Auto-selects single running device
	if got, err := c.ResolveDevice(nil); err != nil || got != "emulator-5554" {
		t.Fatalf("ResolveDevice(nil) = %q, %v; want emulator-5554", got, err)
	}
	// Selects by index 1
	if got, err := c.ResolveDevice([]string{"1"}); err != nil || got != "emulator-5554" {
		t.Fatalf("ResolveDevice([1]) = %q, %v; want emulator-5554", got, err)
	}
	// Selects by exact serial
	if got, err := c.ResolveDevice([]string{"emulator-5554"}); err != nil || got != "emulator-5554" {
		t.Fatalf("ResolveDevice([emulator-5554]) = %q, %v; want emulator-5554", got, err)
	}
	// Selects by AVD name
	if got, err := c.ResolveDevice([]string{"Slim_Pixel_5"}); err != nil || got != "emulator-5554" {
		t.Fatalf("ResolveDevice([Slim_Pixel_5]) = %q, %v; want emulator-5554", got, err)
	}
	// Index out of range
	if _, err := c.ResolveDevice([]string{"2"}); err == nil {
		t.Fatal("expected error for index 2 when only 1 device is running")
	}
	// Unknown name
	if _, err := c.ResolveDevice([]string{"NonExistent"}); err == nil {
		t.Fatal("expected error for non-existent device")
	}

	// Case 3: Two devices running
	twoDevices := "List of devices attached\nemulator-5554\tdevice\nemulator-5556\tdevice\n"
	if err := os.WriteFile(filepath.Join(dir, "devices"), []byte(twoDevices), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "avd_emulator-5556"), []byte("Pixel_10_Pro\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Index 2 selects second device
	if got, err := c.ResolveDevice([]string{"2"}); err != nil || got != "emulator-5556" {
		t.Fatalf("ResolveDevice([2]) = %q, %v; want emulator-5556", got, err)
	}
	// AVD name selects matching device
	if got, err := c.ResolveDevice([]string{"Pixel_10_Pro"}); err != nil || got != "emulator-5556" {
		t.Fatalf("ResolveDevice([Pixel_10_Pro]) = %q, %v; want emulator-5556", got, err)
	}
}

func TestEnableTarget(t *testing.T) {
	c, dir := newFake(t)

	// Slim the device first
	c.Slim("e", false, nil, nil)
	if got := setting(t, dir, "global", "bluetooth_on"); got != "0" {
		t.Fatalf("after slim bluetooth_on = %q, want 0", got)
	}

	// Re-enable bluetooth via Enable
	actions, err := c.Enable("e", "bluetooth")
	if err != nil {
		t.Fatalf("Enable(bluetooth) error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("expected actions for Enable(bluetooth), got none")
	}
	if got := setting(t, dir, "global", "bluetooth_on"); got != "1" {
		t.Fatalf("after enable bluetooth_on = %q, want 1", got)
	}

	// Re-enable animations via Enable
	actions, err = c.Enable("e", "animations")
	if err != nil {
		t.Fatalf("Enable(animations) error: %v", err)
	}
	if got := setting(t, dir, "global", "window_animation_scale"); got != "1" {
		t.Fatalf("after enable window_animation_scale = %q, want 1", got)
	}

	// Re-enable an app alias (maps)
	actions, err = c.Enable("e", "maps")
	if err != nil {
		t.Fatalf("Enable(maps) error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("expected package enabled for maps, got none")
	}
}
