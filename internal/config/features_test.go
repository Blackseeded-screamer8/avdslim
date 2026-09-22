package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestAvd creates a fake ~/.android/avd tree and points ANDROID_AVD_HOME at it.
func writeTestAvd(t *testing.T, name, contents string) string {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, name+".avd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(cfg, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_AVD_HOME", base)
	return cfg
}

// Without config.ini.bak the change cannot be undone, so no backup, no change
// (the same rule TuneAvd follows).
func TestSetHostFeatureRefusesWithoutBackup(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	const orig = "hw.audioOutput=no\nhw.ramSize=1536\n"
	cfg := writeTestAvd(t, "Pixel_Test", orig)
	dir := filepath.Dir(cfg)
	if err := os.Chmod(dir, 0555); err != nil { // config.ini stays writable; .bak cannot be created
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	if _, err := SetHostFeature("Pixel_Test", "audio", true); err == nil {
		t.Fatal("changed config.ini without being able to back it up")
	}
	if b, _ := os.ReadFile(cfg); string(b) != orig {
		t.Errorf("config.ini changed to %q", b)
	}
}

func TestSetHostFeatureRoundTrip(t *testing.T) {
	cfg := writeTestAvd(t, "Pixel_Test", "hw.audioOutput=no\nhw.audioInput=no\nhw.ramSize=1536\n")

	if HostFeatureOn("Pixel_Test", "audio") {
		t.Fatal("audio should start disabled (no marker)")
	}

	if _, err := SetHostFeature("Pixel_Test", "audio", true); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !HostFeatureOn("Pixel_Test", "audio") {
		t.Fatal("audio should be on after enable")
	}

	kv, err := readConfigIni(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if kv["hw.audioOutput"] != "yes" || kv["hw.audioInput"] != "yes" {
		t.Fatalf("hw keys not flipped: %v", kv)
	}
	if kv["hw.ramSize"] != "1536" {
		t.Fatalf("unrelated key lost: %v", kv)
	}
	if _, err := os.Stat(cfg + ".bak"); err != nil {
		t.Fatalf("backup not created: %v", err)
	}

	if _, err := SetHostFeature("Pixel_Test", "audio", false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if HostFeatureOn("Pixel_Test", "audio") {
		t.Fatal("audio should be off after disable")
	}
}

// TuneAvd must not re-mute a feature the user enabled.
func TestApplySlimHardwareHonorsMarkers(t *testing.T) {
	kv := map[string]string{"avdslim.audio": "yes"}
	ApplySlimHardware(kv)

	if kv["hw.audioOutput"] != "yes" {
		t.Fatalf("marked audio was re-muted: %v", kv)
	}
	if kv["hw.camera.back"] != "none" {
		t.Fatalf("unmarked camera should stay slimmed: %v", kv)
	}
	if kv["hw.dPad"] != "no" {
		t.Fatalf("unmarked dpad should stay slimmed: %v", kv)
	}
}

// Every host feature and alias must appear in the user-facing reference.
func TestFeatureDocListsHostFeatures(t *testing.T) {
	data, err := os.ReadFile("../../docs/FEATURES.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)

	for name, f := range hostFeatures {
		if !strings.Contains(doc, "`"+name+"`") {
			t.Errorf("docs/FEATURES.md does not list host feature %q", name)
		}
		for k := range f.hwOn {
			if !strings.Contains(doc, k) {
				t.Errorf("docs/FEATURES.md does not list config key %q (%s)", k, name)
			}
		}
	}
	for alias := range hostFeatureAliases {
		if !strings.Contains(doc, "`"+alias+"`") {
			t.Errorf("docs/FEATURES.md does not list alias %q", alias)
		}
	}
}

func TestHostFeatureNameAliases(t *testing.T) {
	for alias, want := range map[string]string{"sound": "audio", "MIC": "audio", "webcam": "camera", "boot-anim": "bootanim"} {
		got, ok := HostFeatureName(alias)
		if !ok || got != want {
			t.Fatalf("HostFeatureName(%q) = %q, %v; want %q", alias, got, ok, want)
		}
	}
	if _, ok := HostFeatureName("maps"); ok {
		t.Fatal("package alias must not resolve as a host feature")
	}
}
