package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kdbhalala/avdslim/internal/adbtest"
)

// TestMain lets the test binary act as avdslim: run() re-executes it with
// AVDSLIM_RUN_MAIN=1, so each command runs the real main() in its own process.
func TestMain(m *testing.M) {
	if os.Getenv("AVDSLIM_RUN_MAIN") == "1" {
		os.Args = append([]string{"avdslim"}, strings.Fields(os.Getenv("AVDSLIM_ARGS"))...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// serial uses a console port no real emulator is likely to hold, so host
// memory probes (lsof on the port) find nothing.
const serial = "emulator-5990"

// device is a fake guest: a dir holding the fake adb and its state files,
// plus an isolated HOME shared by every run() of the test.
type device struct {
	t         *testing.T
	dir, home string
	env       []string
}

func newDevice(t *testing.T, running bool) *device {
	if runtime.GOOS == "windows" {
		t.Skip("fake adb is a bash script")
	}
	d := &device{t: t, dir: t.TempDir(), home: t.TempDir()}
	if _, err := adbtest.Install(d.dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d.home, "avd"), 0755); err != nil {
		t.Fatal(err)
	}
	if running {
		d.write("devices", "List of devices attached\n"+serial+"\tdevice\n")
		d.write("avd_"+serial, "Slim_Pixel_5\n")
		d.write("packages", "com.google.android.apps.maps\ncom.google.android.youtube\ncom.android.chrome\n")
	}
	return d
}

func (d *device) write(name, content string) {
	if err := os.WriteFile(filepath.Join(d.dir, name), []byte(content), 0644); err != nil {
		d.t.Fatal(err)
	}
}

func (d *device) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(d.dir, name))
	return string(b)
}

// run executes `avdslim args...` with the fake adb first on PATH and HOME
// isolated, returning combined output and whether it exited 0.
func (d *device) run(args ...string) (string, bool) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append([]string{
		"AVDSLIM_RUN_MAIN=1",
		"AVDSLIM_ARGS=" + strings.Join(args, " "),
		"PATH=" + d.dir + string(os.PathListSeparator) + "/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + d.home,
		"ANDROID_AVD_HOME=" + filepath.Join(d.home, "avd"),
	}, d.env...)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q:\n%s", w, out)
		}
	}
}

func TestOnOffRoundTrip(t *testing.T) {
	d := newDevice(t, true)

	out, ok := d.run("on")
	if !ok {
		t.Fatalf("on failed:\n%s", out)
	}
	mustContain(t, out, "Slimming complete for "+serial, "Disabled: com.google.android.apps.maps", "Successfully disabled 2 packages")
	if !strings.Contains(d.read("state"), "com.google.android.apps.maps") {
		t.Fatalf("state missing disabled package: %q", d.read("state"))
	}
	if d.read("global_window_animation_scale") != "" {
		t.Error("animations changed though skipped by default")
	}
	if d.read("global_bluetooth_on") != "0" {
		t.Error("bluetooth not turned off")
	}

	out, ok = d.run("list")
	if !ok {
		t.Fatalf("list failed:\n%s", out)
	}
	mustContain(t, out, serial, "SLIMMED")

	out, ok = d.run("off")
	if !ok {
		t.Fatalf("off failed:\n%s", out)
	}
	mustContain(t, out, "Restored 2 packages", "Successfully restored "+serial)
	if d.read("state") != "" {
		t.Error("state file not removed")
	}
	if d.read("global_bluetooth_on") != "" {
		t.Errorf("bluetooth_on = %q, want unset (its original)", d.read("global_bluetooth_on"))
	}
}

func TestOnFlags(t *testing.T) {
	d := newDevice(t, true)
	out, ok := d.run("on", "--no-anim", "--skip=bluetooth,sync", "--keep=com.google.android.youtube")
	if !ok {
		t.Fatalf("on failed:\n%s", out)
	}
	mustContain(t, out, "Left unchanged (--skip): bluetooth, sync", "Successfully disabled 1 packages")
	if d.read("global_window_animation_scale") != "0" {
		t.Error("--no-anim did not zero animations")
	}
	if d.read("global_bluetooth_on") != "" || d.read("global_auto_sync") != "" {
		t.Error("skipped groups were changed")
	}
	if strings.Contains(d.read("calls"), "disable-user --user 0 com.google.android.youtube") {
		t.Error("--keep package was disabled")
	}
}

func TestOnRejectsUnknownSkip(t *testing.T) {
	d := newDevice(t, true)
	out, ok := d.run("on", "--skip=bogus")
	if ok {
		t.Fatalf("expected non-zero exit:\n%s", out)
	}
	mustContain(t, out, `Unknown --skip value "bogus"`)
	if strings.Contains(d.read("calls"), "disable-user") {
		t.Error("packages disabled despite bad flag")
	}
}

func TestNoEmulator(t *testing.T) {
	d := newDevice(t, false)
	for _, cmd := range []string{"on", "off", "measure"} {
		out, _ := d.run(cmd)
		mustContain(t, out, "no running Android emulator")
	}
	out, _ := d.run("enable", "maps")
	mustContain(t, out, "No running Android emulators")
}

func TestEnableUpdatesState(t *testing.T) {
	d := newDevice(t, true)
	if out, ok := d.run("on"); !ok {
		t.Fatalf("on failed:\n%s", out)
	}
	out, ok := d.run("enable", "maps")
	if !ok {
		t.Fatalf("enable failed:\n%s", out)
	}
	state := d.read("state")
	if strings.Contains(state, "com.google.android.apps.maps") {
		t.Errorf("maps still recorded as disabled: %s", state)
	}
	if !strings.Contains(state, "com.google.android.youtube") {
		t.Errorf("other disabled packages lost: %s", state)
	}
}

func TestDoctor(t *testing.T) {
	d := newDevice(t, false)
	out, ok := d.run("doctor")
	if !ok {
		t.Fatalf("doctor failed:\n%s", out)
	}
	mustContain(t, out, "ADB: "+filepath.Join(d.dir, "adb"), "No active emulators running")

	d = newDevice(t, true)
	d.run("on")
	out, _ = d.run("doctor")
	mustContain(t, out, serial, "Status: ⚡ Slimmed")
}

// fakeAvdmanager handles `create avd -n NAME -k PKG -d DEV` like the real one:
// it writes NAME.avd/config.ini under ANDROID_AVD_HOME with stock values.
const fakeAvdmanager = `#!/bin/bash
echo "$*" >> "$(dirname "$0")/avdmanager_calls"
cat > /dev/null # consume the hardware-profile answer
while [ $# -gt 0 ]; do
  case "$1" in -n) n="$2" ;; -k) k="$2" ;; -d) dev="$2" ;; esac
  shift
done
[ "$dev" = "bogus" ] && { echo "Error: No device found matching --device bogus" >&2; exit 1; }
mkdir -p "$ANDROID_AVD_HOME/$n.avd"
printf 'hw.ramSize=4096\nhw.gpu.mode=auto\nimage.sysdir.1=%s\n' "$k" > "$ANDROID_AVD_HOME/$n.avd/config.ini"
printf 'path=%s\n' "$ANDROID_AVD_HOME/$n.avd" > "$ANDROID_AVD_HOME/$n.ini"
`

// withSdk gives the device an SDK holding the given system images (paths
// under system-images/, "ABI" replaced by the host ABI) and a fake avdmanager.
func (d *device) withSdk(images ...string) {
	sdk := d.t.TempDir()
	abi := "x86_64"
	if runtime.GOARCH == "arm64" {
		abi = "arm64-v8a"
	}
	for _, img := range images {
		if err := os.MkdirAll(filepath.Join(sdk, "system-images", strings.ReplaceAll(img, "ABI", abi)), 0755); err != nil {
			d.t.Fatal(err)
		}
	}
	d.write("avdmanager", fakeAvdmanager)
	if err := os.Chmod(filepath.Join(d.dir, "avdmanager"), 0755); err != nil {
		d.t.Fatal(err)
	}
	d.env = append(d.env, "ANDROID_HOME="+sdk)
}

func TestCreate(t *testing.T) {
	d := newDevice(t, false)
	d.withSdk("android-34/google_apis/ABI", "android-35/google_apis/ABI", "android-36/google_apis_ps16k/ABI")

	out, ok := d.run("create", "Slim_Test")
	if !ok {
		t.Fatalf("create failed:\n%s", out)
	}
	mustContain(t, out, "android-35;google_apis", "Successfully tuned AVD \"Slim_Test\"", "avdslim bake Slim_Test")
	calls := d.read("avdmanager_calls")
	mustContain(t, calls, "create avd -n Slim_Test", ";android-35;google_apis;", "-d pixel_5")

	cfg, _ := os.ReadFile(filepath.Join(d.home, "avd", "Slim_Test.avd", "config.ini"))
	mustContain(t, string(cfg), "hw.ramSize=1536")
	if _, err := os.Stat(filepath.Join(d.home, "avd", "Slim_Test.avd", "config.ini.bak")); err != nil {
		t.Error("no config.ini.bak backup of avdmanager's stock config")
	}

	// A second create must not overwrite the existing AVD.
	out, ok = d.run("create", "slim_test")
	if ok {
		t.Fatalf("create over existing AVD succeeded:\n%s", out)
	}
	mustContain(t, out, "already exists")
	if n := strings.Count(d.read("avdmanager_calls"), "\n"); n != 1 {
		t.Errorf("avdmanager called %d times, want 1", n)
	}
}

func TestCreateOptions(t *testing.T) {
	d := newDevice(t, false)
	d.withSdk("android-34/google_apis/ABI", "android-35/google_apis/ABI")

	out, ok := d.run("create", "Old", "--api=34", "--device=pixel_6", "--ram=2048")
	if !ok {
		t.Fatalf("create failed:\n%s", out)
	}
	mustContain(t, d.read("avdmanager_calls"), ";android-34;", "-d pixel_6")
	cfg, _ := os.ReadFile(filepath.Join(d.home, "avd", "Old.avd", "config.ini"))
	mustContain(t, string(cfg), "hw.ramSize=2048")
}

func TestCreateFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		images []string
		args   []string
		want   string
	}{
		{"no name", []string{"android-35/google_apis/ABI"}, nil, "Please give the new AVD a name"},
		{"bad name", []string{"android-35/google_apis/ABI"}, []string{"my/avd"}, "Please give the new AVD a name"},
		{"bad api", []string{"android-35/google_apis/ABI"}, []string{"X", "--api=abc"}, "Invalid --api=abc"},
		{"only 16k and playstore", []string{"android-36/google_apis_ps16k/ABI", "android-36/google_apis_playstore/ABI"}, []string{"X"}, `sdkmanager "system-images;android-35;google_apis;`},
		{"api not installed", []string{"android-35/google_apis/ABI"}, []string{"X", "--api=33"}, `sdkmanager "system-images;android-33;google_apis;`},
		{"avdmanager fails", []string{"android-35/google_apis/ABI"}, []string{"X", "--device=bogus"}, "avdmanager failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newDevice(t, false)
			d.withSdk(tc.images...)
			out, ok := d.run(append([]string{"create"}, tc.args...)...)
			if ok {
				t.Fatalf("expected non-zero exit:\n%s", out)
			}
			mustContain(t, out, tc.want)
			if entries, _ := os.ReadDir(filepath.Join(d.home, "avd")); len(entries) > 0 && tc.name != "avdmanager fails" {
				t.Errorf("AVD dir not empty after failure: %v", entries)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	d := newDevice(t, false)
	out, ok := d.run("frobnicate")
	if ok {
		t.Fatal("expected non-zero exit")
	}
	mustContain(t, out, "Unknown command: frobnicate")
}
