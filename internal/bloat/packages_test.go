package bloat

import (
	"regexp"
	"testing"
)

func allBloat() []string {
	all := append([]string{}, AggressiveBloatPackages...)
	for _, pkgs := range StandardBloatCategories {
		all = append(all, pkgs...)
	}
	return all
}

// protected are packages the README promises stay working (GMS/FCM/Auth,
// WebView, photo picker) or without which the emulator is unusable. Adding
// one to a bloat list must be a deliberate decision that edits this test.
var protected = []string{
	"com.google.android.gms",
	"com.google.android.gsf",
	"com.google.android.webview",
	"com.android.webview",
	"com.google.android.trichromelibrary",
	"com.google.android.providers.media.module",
	"com.android.providers.media",
	"com.google.android.permissioncontroller",
	"com.google.android.packageinstaller",
	"com.android.systemui",
	"com.android.settings",
	"com.android.shell",
	"com.android.phone",
	"com.google.android.apps.nexuslauncher",
	"com.android.launcher3",
	"android",
}

func TestNoBootCriticalOrProtectedPackages(t *testing.T) {
	never := append(append([]string{}, BootCritical...), protected...)
	for _, p := range allBloat() {
		for _, c := range never {
			if p == c {
				t.Errorf("bloat list contains %s, which must never be disabled", c)
			}
		}
	}
}

func TestBloatListsWellFormed(t *testing.T) {
	valid := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)
	seen := map[string]bool{}
	for _, p := range allBloat() {
		if !valid.MatchString(p) {
			t.Errorf("malformed package name %q", p)
		}
		if seen[p] {
			t.Errorf("%s listed twice", p)
		}
		seen[p] = true
	}
}
