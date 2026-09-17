package config

import (
	"os"
	"path/filepath"
	"testing"
)

// A failed backup must leave config.ini and snapshots untouched.
func TestTuneAvdAbortsWhenBackupFails(t *testing.T) {
	base := t.TempDir()
	avdDir := filepath.Join(base, "Test_Avd.avd")
	if err := os.MkdirAll(filepath.Join(avdDir, "snapshots", "avdslim_clean"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(avdDir, "config.ini")
	original := "hw.ramSize=4096\n"
	if err := os.WriteFile(cfg, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	// A read-only AVD dir makes the backup write fail with EACCES.
	if err := os.Chmod(avdDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(avdDir, 0755) })
	t.Setenv("ANDROID_AVD_HOME", base)

	if err := TuneAvd("Test_Avd", 1536, 256, "host"); err == nil {
		t.Fatal("TuneAvd succeeded despite unwritable backup")
	}
	if got, _ := os.ReadFile(cfg); string(got) != original {
		t.Errorf("config.ini rewritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(avdDir, "snapshots", "avdslim_clean")); err != nil {
		t.Errorf("snapshots purged: %v", err)
	}
}
