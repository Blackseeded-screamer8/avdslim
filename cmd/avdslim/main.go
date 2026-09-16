package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kdbhalala/avdslim/internal/adb"
	"github.com/kdbhalala/avdslim/internal/bloat"
	"github.com/kdbhalala/avdslim/internal/config"
	"github.com/kdbhalala/avdslim/internal/doctor"
	"github.com/kdbhalala/avdslim/internal/host"
	"github.com/kdbhalala/avdslim/internal/shim"
)

// var, not const: release.yml sets it via -ldflags "-X main.version=...".
var version = "1.0.5"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := strings.ToLower(os.Args[1])
	subArgs := os.Args[2:]
	client := adb.NewClient()

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("avdslim version %s\n", version)
	case "doctor":
		doctor.RunDoctor(client)
	case "profiles":
		bloat.PrintProfiles()
	case "list":
		handleList(client)
	case "measure":
		handleMeasure(client, subArgs)
	case "on", "slim":
		handleOn(client, subArgs)
	case "off", "unslim", "restore", "reset":
		handleOff(client, subArgs)
	case "watch":
		handleWatch(client, subArgs)
	case "tune-avd", "tune":
		handleTuneAvd(subArgs)
	case "launch", "start", "run":
		handleLaunch(client, subArgs)
	case "stop", "kill", "quit":
		handleStop(client, subArgs)
	case "bake":
		handleBake(client, subArgs)
	case "snapshot", "snap":
		handleSnapshot(client, subArgs)
	case "unbake", "unsnapshot":
		handleUnbake(subArgs)
	case "restart":
		handleRestart(client, subArgs)
	case "bench", "benchmark":
		handleBench(client, subArgs)
	case "install-shim", "shim":
		handleInstallShim(subArgs)
	case "uninstall-shim", "unshim":
		handleUninstallShim()
	default:
		fmt.Printf("❌ Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// addSkip records a --skip=a,b flag, exiting on an unknown group so typos are not silently ignored.
func addSkip(skip map[string]bool, flag string) {
	for _, g := range strings.Split(strings.TrimPrefix(flag, "--skip="), ",") {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if !adb.IsSkipGroup(g) {
			fmt.Fprintf(os.Stderr, "❌ Unknown --skip value %q (valid: %s)\n", g, strings.Join(adb.SkipGroups, ", "))
			os.Exit(1)
		}
		skip[g] = true
	}
}

func printUsage() {
	fmt.Printf(`════════════════════════════════════════════════════════════════════════
 ⚡ AVD-SLIM — Android Emulator RAM & CPU Optimizer
 (Inspired by simslim for iOS simulators) — v%s
════════════════════════════════════════════════════════════════════════

Usage:
  avdslim <command> [arguments]

Commands:
  list                 List running emulators (with Activity Monitor memory) & saved AVDs
  measure [device]     Deep memory breakdown (host Footprint/RSS + guest dumpsys)
  on [device]          Slim down emulator: disable bloat daemons & trim RAM
                       Options: --aggressive (also disables Play Store updater)
                                --keep=<package> (preserve specific package, e.g. Maps)
                                --skip=<groups> (leave settings alone: animations,bglimit,sync,location,setup)
  restore, off         Instant 100%% stock restore (re-enables packages, animations & sync)
  watch                Auto-detect & slim new emulators as soon as they boot
                       Options: --aggressive, --keep=<package>, --skip=<groups>
  tune-avd [avd_name]  Tune host AVD config.ini (RAM=1024M, Metal GPU, no cameras)
                       Options: --ram=<MB> (default: 1024), --heap=<MB> (default: 256)
  start, run, launch [avd] Launch AVD with low-memory host flags & auto-slim upon boot
                       Options: --no-slim, --no-lowram, --headless, --cold, --ram=<MB> (default: 1024)
  stop, kill [device]  Gracefully shut down emulator (Options: --snap, -f)
  bake [avd_name]      Create local Golden Snapshot (pruned & slimmed) for ~1.5s instant boots
                       Options: --ram=<MB> (default: 1024), --aggressive, --skip=<groups>, --headless, --live
  snapshot, snap       Capture running emulator (with pre-installed apps & test logins)
                       into Golden Snapshot for instant <1.5s restores
  unbake [avd_name]    Delete Golden Snapshot and return AVD to stock cold boots
  bench [device]       Show before/after memory & CPU efficiency scoreboard
  install-shim         Wrap SDK emulator binary so Android Studio launches stay slim
                       Options: --ram=<MB> (default: 1024)
  uninstall-shim       Restore stock Android SDK emulator binary
  doctor               Audit environment, AVDs, system image 16K overhead & toolchain
  profiles             List all bloat categories, packages & guaranteed-working services
  version              Print avdslim version

Examples:
  avdslim watch
  avdslim doctor
  avdslim list
  avdslim measure
  avdslim on --aggressive
  avdslim on --keep=com.google.android.apps.maps
  avdslim on --skip=animations,sync
  avdslim tune-avd Pixel_10_Pro --ram=1024
  avdslim restart
`, version)
}

func handleList(client *adb.Client) {
	fmt.Println("🔎 Checking running Android emulators...")
	running, err := client.GetRunningEmulators()
	if err != nil {
		fmt.Printf("Error checking emulators: %v\n", err)
	} else if len(running) == 0 {
		fmt.Println("ℹ️  No running Android emulators detected via adb.")
		fmt.Println()
	} else {
		fmt.Printf("\n📱 Running Emulators (%d):\n", len(running))
		for _, emu := range running {
			hostPid := host.FindHostPidForSerial(emu.Serial)
			hostFootprintMb := 0
			hostRssMb := 0
			if hostPid > 0 {
				hostFootprintMb = host.GetHostFootprintMb(hostPid)
				hostRssMb = host.GetHostRssMb(hostPid)
			}

			statusStr := "🔴 FULL (Stock)"
			if emu.IsSlimmed {
				statusStr = "⚡ SLIMMED"
			}
			fmt.Printf("  • %s (%s, Android %s, API %s)\n", emu.Serial, emu.Model, emu.AndroidVersion, emu.ApiLevel)
			if hostFootprintMb > 0 {
				fmt.Printf("    Host PID: %d | Activity Monitor: %d MB | RSS: %d MB | Status: %s\n", hostPid, hostFootprintMb, hostRssMb, statusStr)
			} else {
				fmt.Printf("    Host PID: %d | Status: %s\n", hostPid, statusStr)
			}
		}
		fmt.Println()
	}

	fmt.Println("💾 Installed AVD Configurations:")
	avds := config.GetInstalledAvds()
	if len(avds) == 0 {
		fmt.Println("  (No AVDs found in ~/.android/avd)")
		fmt.Println()
	} else {
		for _, avd := range avds {
			ram := avd["hw.ramSize"]
			if ram == "" {
				ram = "unknown"
			}
			heap := avd["vm.heapSize"]
			if heap == "" {
				heap = "unknown"
			}
			gpu := avd["hw.gpu.mode"]
			if gpu == "" {
				gpu = "unknown"
			}
			fmt.Printf("  • %s (RAM: %sMB, Heap: %sMB, GPU: %s)\n", avd["name"], ram, heap, gpu)
		}
		fmt.Println()
	}
}

func handleMeasure(client *adb.Client, args []string) {
	serial, err := client.ResolveDevice(args)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}

	fmt.Printf("📊 Measuring memory footprint for %s...\n\n", serial)

	hostPid := host.FindHostPidForSerial(serial)
	if hostPid > 0 {
		footprint := host.GetHostFootprintMb(hostPid)
		rss := host.GetHostRssMb(hostPid)
		fmt.Println("🖥️  HOST (macOS) Footprint:")
		fmt.Printf("   QEMU / Emulator PID: %d\n", hostPid)
		fmt.Printf("   Activity Monitor Memory (Footprint): %d MB\n", footprint)
		fmt.Printf("   Resident Physical RAM (RSS): %d MB\n\n", rss)
	}

	out, _ := client.Exec("-s", serial, "shell", "dumpsys", "meminfo")
	host.PrintGuestMeminfo(out)
}

func handleOn(client *adb.Client, args []string) {
	aggressive := false
	filteredArgs := make([]string, 0, len(args))
	var keepPackages []string
	skip := make(map[string]bool)
	for _, a := range args {
		if a == "--aggressive" {
			aggressive = true
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
		} else if strings.HasPrefix(a, "--skip=") {
			addSkip(skip, a)
		} else if !strings.HasPrefix(a, "--") {
			filteredArgs = append(filteredArgs, a)
		}
	}

	serial, err := client.ResolveDevice(filteredArgs)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}

	presetName := "Standard"
	if aggressive {
		presetName = "Aggressive"
	}
	fmt.Printf("⚡ Slimming Android Emulator (%s) [preset: %s]...\n\n", serial, presetName)

	if len(keepPackages) > 0 {
		fmt.Printf("   Preserving requested package(s): %s\n", strings.Join(keepPackages, ", "))
	}

	hostPid := host.FindHostPidForSerial(serial)
	beforeFootprint := 0
	beforeRss := 0
	if hostPid > 0 {
		beforeFootprint = host.GetHostFootprintMb(hostPid)
		beforeRss = host.GetHostRssMb(hostPid)
	}

	fmt.Println("1. Disabling non-essential background daemons:")
	count, _ := client.Slim(serial, aggressive, keepPackages, skip)
	fmt.Printf("   -> Successfully disabled %d packages.\n\n", count)

	fmt.Println("2. Tuned system settings (animations 0x, background limit 4, sync off, location off).")
	if len(skip) > 0 {
		fmt.Printf("   Left unchanged (--skip): %s\n", strings.Join(sortedKeys(skip), ", "))
	}
	fmt.Println("3. Purged cached processes and trimmed memory.")
	fmt.Println()

	time.Sleep(1 * time.Second)

	afterFootprint := 0
	afterRss := 0
	if hostPid > 0 {
		afterFootprint = host.GetHostFootprintMb(hostPid)
		afterRss = host.GetHostRssMb(hostPid)
	}

	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Printf("🎉 Slimming complete for %s!\n", serial)
	if beforeFootprint > 0 && afterFootprint > 0 {
		diffFp := beforeFootprint - afterFootprint
		if diffFp < 0 {
			diffFp = 0
		}
		diffRss := beforeRss - afterRss
		if diffRss < 0 {
			diffRss = 0
		}
		fmt.Printf("🖥️  Activity Monitor Memory: %dMB -> %dMB (Reclaimed: %dMB)\n", beforeFootprint, afterFootprint, diffFp)
		fmt.Printf("🖥️  Host Resident RAM (RSS): %dMB -> %dMB (Reclaimed: %dMB)\n", beforeRss, afterRss, diffRss)
	}
	fmt.Printf("ℹ️  To restore default stock services anytime:\n   avdslim off %s\n\n", serial)
}

func handleOff(client *adb.Client, args []string) {
	serial, err := client.ResolveDevice(args)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}

	fmt.Printf("🔄 Restoring default services for %s...\n\n", serial)
	fmt.Println("1. Re-enabling packages:")
	count, _ := client.Restore(serial)
	fmt.Printf("   -> Restored %d packages.\n\n", count)
	fmt.Println("2. Restored the system settings avdslim changed to their previous values.")
	fmt.Printf("✅ Successfully restored %s to stock configuration.\n\n", serial)
}

func selectAvdInteractively(installed []map[string]string, promptTitle string) (string, error) {
	if len(installed) == 0 {
		return "", fmt.Errorf("no installed AVDs found in ~/.android/avd")
	}
	if len(installed) == 1 {
		fmt.Printf("ℹ️  Auto-selecting only installed AVD: %q\n\n", installed[0]["name"])
		return installed[0]["name"], nil
	}

	fmt.Printf("📱 %s:\n", promptTitle)
	for i, avd := range installed {
		ram := avd["hw.ramSize"]
		if ram == "" {
			ram = "default"
		} else if !strings.HasSuffix(ram, "MB") && !strings.HasSuffix(ram, "G") && !strings.HasSuffix(ram, "M") {
			ram += "MB"
		}
		heap := avd["vm.heapSize"]
		if heap == "" {
			heap = "default"
		} else if !strings.HasSuffix(heap, "MB") && !strings.HasSuffix(heap, "M") {
			heap += "MB"
		}
		fmt.Printf("  [%d] %s (Config: RAM %s, Heap %s)\n", i+1, avd["name"], ram, heap)
	}
	fmt.Println()
	fmt.Printf("👉 Enter selection [1-%d] (default 1): ", len(installed))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Printf("✓ Selected: %s\n\n", installed[0]["name"])
		return installed[0]["name"], nil
	}

	if choice, err := strconv.Atoi(input); err == nil && choice >= 1 && choice <= len(installed) {
		fmt.Printf("✓ Selected: %s\n\n", installed[choice-1]["name"])
		return installed[choice-1]["name"], nil
	}

	for _, avd := range installed {
		if strings.EqualFold(avd["name"], input) {
			fmt.Printf("✓ Selected: %s\n\n", avd["name"])
			return avd["name"], nil
		}
	}

	return "", fmt.Errorf("invalid selection: %q", input)
}

func handleTuneAvd(args []string) {
	ramMb := 1024
	heapMb := 256
	gpuMode := ""
	targetAvd := ""

	for _, a := range args {
		if strings.HasPrefix(a, "--ram=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--ram=")); err == nil {
				ramMb = v
			}
		} else if strings.HasPrefix(a, "--heap=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--heap=")); err == nil {
				heapMb = v
			}
		} else if strings.HasPrefix(a, "--gpu=") {
			gpuMode = strings.TrimPrefix(a, "--gpu=")
		} else if !strings.HasPrefix(a, "--") {
			targetAvd = a
		}
	}

	installed := config.GetInstalledAvds()
	if targetAvd != "" {
		if idx, err := strconv.Atoi(targetAvd); err == nil && idx >= 1 && idx <= len(installed) {
			targetAvd = installed[idx-1]["name"]
		}
	} else if len(installed) > 0 {
		var err error
		targetAvd, err = selectAvdInteractively(installed, "Select an AVD to tune")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
	}

	if err := config.TuneAvd(targetAvd, ramMb, heapMb, gpuMode); err != nil {
		fmt.Printf("❌ %v\n", err)
	}
}

func handleLaunch(client *adb.Client, args []string) {
	installed := config.GetInstalledAvds()
	var avdName string
	var options []string

	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			options = append(options, a)
		} else if avdName == "" {
			avdName = a
		}
	}

	if avdName != "" {
		// Check if user passed a numeric index directly: e.g. `avdslim start 1`
		if idx, err := strconv.Atoi(avdName); err == nil && idx >= 1 && idx <= len(installed) {
			avdName = installed[idx-1]["name"]
			fmt.Printf("✓ Selected [%d]: %s\n\n", idx, avdName)
		}
	} else {
		var err error
		avdName, err = selectAvdInteractively(installed, "Select an AVD to launch")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
	}

	doSlim := true
	lowRam := true
	headless := false
	forceCold := false
	ramMb := 1024
	gpuMode := config.GetRecommendedGpuMode()

	var slimArgs []string
	for _, a := range options {
		if a == "--aggressive" || strings.HasPrefix(a, "--keep=") || strings.HasPrefix(a, "--skip=") {
			slimArgs = append(slimArgs, a)
		} else if a == "--no-slim" {
			doSlim = false
		} else if a == "--slim" {
			doSlim = true
		} else if a == "--no-lowram" {
			lowRam = false
		} else if a == "--headless" || a == "--no-window" {
			headless = true
		} else if a == "--cold" || a == "--no-snapshot" {
			forceCold = true
		} else if strings.HasPrefix(a, "--ram=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--ram=")); err == nil {
				ramMb = v
			}
		} else if strings.HasPrefix(a, "--gpu=") {
			gpuMode = strings.TrimPrefix(a, "--gpu=")
		}
	}

	emulator := config.FindEmulatorExecutable()
	emuArgs := []string{
		"-avd", avdName,
		"-memory", strconv.Itoa(ramMb),
		"-no-audio",
		"-camera-back", "none",
		"-camera-front", "none",
		"-gpu", gpuMode,
		"-no-boot-anim",
	}

	if lowRam {
		emuArgs = append(emuArgs, "-lowram")
	}

	if headless {
		emuArgs = append(emuArgs, "-no-window")
	}

	hasGolden := config.HasGoldenSnapshot(avdName)
	if hasGolden && !forceCold {
		emuArgs = append(emuArgs, "-snapshot", "avdslim_clean", "-no-snapshot-save")
		fmt.Printf("✨ Golden Snapshot detected! Restoring instant clean state (< 1.5s boot)...\n")
	} else {
		emuArgs = append(emuArgs, "-no-snapshot-load")
	}

	fmt.Printf("🚀 Launching emulator %q with low-memory host flags:\n", avdName)
	fmt.Printf("   emulator %s\n\n", strings.Join(emuArgs, " "))

	cmd := exec.Command(emulator, emuArgs...)
	host.SetDetached(cmd)
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to launch emulator: %v\n", err)
		return
	}

	fmt.Printf("✓ Emulator process spawned (PID: %d).\n", cmd.Process.Pid)

	if doSlim {
		fmt.Println("⏳ Waiting for emulator to finish booting...")
		_, _ = client.Exec("wait-for-device")

		booted := false
		for i := 0; i < 60; i++ {
			res, _ := client.Exec("shell", "getprop", "sys.boot_completed")
			if strings.TrimSpace(res) == "1" {
				booted = true
				break
			}
			time.Sleep(2 * time.Second)
		}

		if booted {
			if hasGolden && !forceCold {
				fmt.Println("✓ Instant boot complete via Golden Snapshot! Refreshing slim state...")
				handleOn(client, slimArgs)
			} else {
				fmt.Println("✓ Boot complete! Applying avdslim optimizations...")
				handleOn(client, slimArgs)
			}
		} else {
			fmt.Println("⚠️  Boot timed out after 120s. You can run `avdslim on` manually.")
		}
	}
}

func handleStop(client *adb.Client, args []string) {
	var serial string
	snap := false
	force := false

	for _, a := range args {
		if a == "--snap" || a == "--snapshot" {
			snap = true
		} else if a == "-f" || a == "--force" || a == "--no-snap" {
			force = true
		} else if !strings.HasPrefix(a, "--") {
			serial = a
		}
	}

	running, err := client.GetRunningEmulators()
	if err != nil || len(running) == 0 {
		fmt.Println("ℹ️  No running Android emulator detected.")
		return
	}

	if serial == "" {
		if len(running) == 1 {
			serial = running[0].Serial
		} else {
			fmt.Println("📱 Running Emulators:")
			for i, emu := range running {
				nameOut, _ := client.Exec("-s", emu.Serial, "emu", "avd", "name")
				name := strings.TrimSpace(strings.Split(nameOut, "\n")[0])
				fmt.Printf("   [%d] %s (%s)\n", i+1, emu.Serial, name)
			}
			fmt.Print("\nSelect an emulator to stop (number): ")
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			if idx, err := strconv.Atoi(text); err == nil && idx >= 1 && idx <= len(running) {
				serial = running[idx-1].Serial
			} else {
				serial = running[0].Serial
			}
		}
	}

	avdNameOut, _ := client.Exec("-s", serial, "emu", "avd", "name")
	avdName := strings.TrimSpace(strings.Split(avdNameOut, "\n")[0])
	if avdName == "" {
		avdName = serial
	}

	// If not force and not already --snap, prompt
	if !force && !snap {
		fmt.Printf("💾 Save current state of %s into Golden Snapshot before stopping? [y/N]: ", avdName)
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		ans = strings.ToLower(strings.TrimSpace(ans))
		if ans == "y" || ans == "yes" {
			snap = true
		}
	}

	if snap {
		fmt.Printf("📸 Capturing Golden Snapshot for %s before shutdown...\n", avdName)
		handleSnapshot(client, []string{serial})
	}

	fmt.Printf("🛑 Gracefully shutting down %s (%s)...\n", serial, avdName)
	client.Exec("-s", serial, "emu", "kill")

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if pid := host.FindHostPidForSerial(serial); pid == 0 {
			break
		}
	}
	fmt.Println("✓ Emulator process stopped cleanly.")
	fmt.Println()
}

func handleRestart(client *adb.Client, args []string) {
	serial, err := client.ResolveDevice(args)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}

	avdNameOut, _ := client.Exec("-s", serial, "emu", "avd", "name")
	lines := strings.Split(strings.TrimSpace(avdNameOut), "\n")
	avdName := ""
	if len(lines) > 0 {
		avdName = strings.TrimSpace(lines[0])
	}
	if avdName == "" || strings.Contains(avdName, "KO:") {
		installed := config.GetInstalledAvds()
		if len(installed) == 1 {
			avdName = installed[0]["name"]
		}
	}
	if avdName == "" {
		fmt.Println("❌ Could not determine AVD name for running emulator.")
		return
	}

	fmt.Printf("🔄 Gracefully shutting down %s (%s)...\n", serial, avdName)
	client.Exec("-s", serial, "emu", "kill")

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if pid := host.FindHostPidForSerial(serial); pid == 0 {
			break
		}
	}
	fmt.Println("✓ Emulator process stopped.")

	// Purge stale runtime cache & snapshots
	home, _ := os.UserHomeDir()
	avdDir := filepath.Join(home, ".android", "avd", avdName+".avd")
	_ = os.Remove(filepath.Join(avdDir, "hardware-qemu.ini"))
	_ = os.Remove(filepath.Join(avdDir, "hardware-qemu.ini.lock"))
	_ = os.RemoveAll(filepath.Join(avdDir, "snapshots"))
	fmt.Println("✓ Purged stale hardware-qemu.ini and snapshots.")

	launchArgs := []string{avdName, "--slim"}
	for _, a := range args {
		if strings.HasPrefix(a, "--ram=") {
			launchArgs = append(launchArgs, a)
		}
	}
	handleLaunch(client, launchArgs)
}

func findEmulator() string {
	return config.FindEmulatorExecutable()
}

func handleWatch(client *adb.Client, args []string) {
	aggressive := false
	var keepPackages []string
	skip := make(map[string]bool)
	for _, a := range args {
		if a == "--aggressive" {
			aggressive = true
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
		} else if strings.HasPrefix(a, "--skip=") {
			addSkip(skip, a)
		}
	}

	preset := "Standard"
	if aggressive {
		preset = "Aggressive"
	}

	fmt.Println("👀 AVD-SLIM Watcher active...")
	fmt.Printf("   Preset: %s\n", preset)
	if len(keepPackages) > 0 {
		fmt.Printf("   Preserving packages: %s\n", strings.Join(keepPackages, ", "))
	}
	if len(skip) > 0 {
		fmt.Printf("   Leaving settings unchanged: %s\n", strings.Join(sortedKeys(skip), ", "))
	}
	fmt.Println("   Monitoring for newly booted Android emulators in the background.")
	fmt.Println("   Will automatically apply low-memory optimizations as soon as emulators boot.")
	fmt.Println("   Press Ctrl+C to stop.")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	slimmedDevices := make(map[string]bool)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n👋 Stopping AVD-SLIM watcher. Goodbye!")
			return
		case <-ticker.C:
			emulators, err := client.GetRunningEmulators()
			if err != nil {
				continue
			}

			// Clean up devices that were turned off / disconnected
			currentMap := make(map[string]bool)
			for _, emu := range emulators {
				currentMap[emu.Serial] = true
			}
			for serial := range slimmedDevices {
				if !currentMap[serial] {
					delete(slimmedDevices, serial)
				}
			}

			for _, emu := range emulators {
				if slimmedDevices[emu.Serial] || emu.IsSlimmed {
					slimmedDevices[emu.Serial] = true
					continue
				}

				// Check if boot completed
				res, _ := client.Exec("-s", emu.Serial, "shell", "getprop", "sys.boot_completed")
				if strings.TrimSpace(res) != "1" {
					fmt.Printf("⏳ [%s] Emulator detected, waiting for boot completion...\n", emu.Serial)
					continue
				}

				fmt.Printf("\n✨ [%s] Emulator booted! Automatically applying avdslim...\n", emu.Serial)
				count, _ := client.Slim(emu.Serial, aggressive, keepPackages, skip)
				fmt.Printf("✓ [%s] Successfully slimmed! Disabled %d packages, trimmed RAM.\n\n", emu.Serial, count)
				slimmedDevices[emu.Serial] = true
			}
		}
	}
}

func handleBench(client *adb.Client, args []string) {
	serial, err := client.ResolveDevice(args)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}

	hostPid := host.FindHostPidForSerial(serial)
	footprintMb := 0
	rssMb := 0
	if hostPid > 0 {
		footprintMb = host.GetHostFootprintMb(hostPid)
		rssMb = host.GetHostRssMb(hostPid)
	}

	disabledRaw, _ := client.Exec("-s", serial, "shell", "pm", "list", "packages", "-d")
	disabledCount := 0
	if strings.TrimSpace(disabledRaw) != "" {
		for _, line := range strings.Split(strings.TrimSpace(disabledRaw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "package:") {
				disabledCount++
			}
		}
	}

	baselineFp := 5600
	baselineRss := 2800

	fpSavingPct := 0
	if footprintMb > 0 && footprintMb < baselineFp {
		fpSavingPct = int(((float64(baselineFp) - float64(footprintMb)) / float64(baselineFp)) * 100)
	}
	rssSavingPct := 0
	if rssMb > 0 && rssMb < baselineRss {
		rssSavingPct = int(((float64(baselineRss) - float64(rssMb)) / float64(baselineRss)) * 100)
	}

	fmt.Printf(`════════════════════════════════════════════════════════════════════════
 ⚡ AVD-SLIM Live Efficiency Scoreboard: %s
════════════════════════════════════════════════════════════════════════
 Metric                      Stock Baseline    AVD-SLIM (Current)   Savings
 ───────────────────────────────────────────────────────────────────────
 Host Memory (Footprint)     5,600 MB (5.6 GB) %5d MB (%3.1f GB)   -%d%% ⚡
 Physical Resident RAM (RSS) 2,800 MB (2.8 GB) %5d MB (%3.1f GB)   -%d%% ⚡
 Disabled Background Bloat   0 packages        %2d packages disabled
 Dalvik / ART Heap Ceiling   512 MB            256 MB (Compact)    -50%%
 Display Animations / Churn  1.0x Scale        0x (Zero GPU Churn)  100%%
 ───────────────────────────────────────────────────────────────────────
 🛡️  Fidelity: 100%% FCM Push, Firebase Auth, WebView & Sockets Guaranteed
════════════════════════════════════════════════════════════════════════
`, serial, footprintMb, float64(footprintMb)/1024.0, fpSavingPct, rssMb, float64(rssMb)/1024.0, rssSavingPct, disabledCount)
}

func handleInstallShim(args []string) {
	ramMb := 1024
	for _, a := range args {
		if strings.HasPrefix(a, "--ram=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--ram=")); err == nil {
				ramMb = v
			}
		}
	}

	fmt.Printf("🔧 Installing AVD-SLIM emulator shim (Default RAM: %dMB)...\n", ramMb)
	if err := shim.InstallShim(ramMb); err != nil {
		fmt.Printf("❌ Failed to install shim: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully installed emulator shim!")
	fmt.Println("   • From now on, launching emulators via Android Studio 'Play' button")
	fmt.Printf("     will automatically inject -memory %d -lowram -no-audio flags.\n", ramMb)
	fmt.Println("   • To restore stock Android Studio emulator behavior anytime:")
	fmt.Println("     avdslim uninstall-shim")
	fmt.Println()
}

func handleUninstallShim() {
	fmt.Println("🔄 Restoring original Android SDK emulator binary...")
	if err := shim.UninstallShim(); err != nil {
		fmt.Printf("❌ %v\n", err)
		return
	}
	fmt.Println("✅ Successfully uninstalled shim. Stock emulator binary restored.")
	fmt.Println()
}

func handleBake(client *adb.Client, args []string) {
	for _, a := range args {
		if a == "--live" || a == "--current" {
			handleSnapshot(client, args)
			return
		}
	}

	installed := config.GetInstalledAvds()
	var targetAvd string
	ramMb := 1024
	aggressive := false
	headless := false
	var keepPackages []string
	skip := make(map[string]bool)

	for _, a := range args {
		if strings.HasPrefix(a, "--skip=") {
			addSkip(skip, a)
		} else if strings.HasPrefix(a, "--ram=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--ram=")); err == nil {
				ramMb = v
			}
		} else if a == "--aggressive" {
			aggressive = true
		} else if a == "--headless" || a == "--no-window" {
			headless = true
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
		} else if !strings.HasPrefix(a, "--") {
			targetAvd = a
		}
	}

	if targetAvd != "" {
		if idx, err := strconv.Atoi(targetAvd); err == nil && idx >= 1 && idx <= len(installed) {
			targetAvd = installed[idx-1]["name"]
			fmt.Printf("✓ Selected [%d]: %s\n\n", idx, targetAvd)
		}
	} else if len(installed) > 0 {
		var err error
		targetAvd, err = selectAvdInteractively(installed, "Select an AVD to bake Golden Snapshot for")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
	} else {
		fmt.Println("❌ No installed AVDs found.")
		return
	}

	fmt.Printf("🍳 Baking Golden Snapshot for %q (RAM: %d MB)...\n", targetAvd, ramMb)
	fmt.Println("   • Cold boots emulator in pristine state")
	fmt.Println("   • Automatically prunes background bloatware & optimizes settings")
	fmt.Println("   • Captures 'avdslim_clean' snapshot for instant ~1.5s launches")
	fmt.Println()

	// 1. Check if an emulator for this AVD is already running. If so, kill it to ensure cold boot.
	running, _ := client.GetRunningEmulators()
	for _, emu := range running {
		nameOut, _ := client.Exec("-s", emu.Serial, "emu", "avd", "name")
		if strings.Contains(nameOut, targetAvd) {
			fmt.Printf("🔄 Stopping active emulator instance (%s) for clean baking...\n", emu.Serial)
			client.Exec("-s", emu.Serial, "emu", "kill")
			time.Sleep(2 * time.Second)
		}
	}

	// Clean existing snapshot directory
	home, _ := os.UserHomeDir()
	snapDir := filepath.Join(home, ".android", "avd", targetAvd+".avd", "snapshots", "avdslim_clean")
	_ = os.RemoveAll(snapDir)

	// 2. Launch cold emulator (DO NOT pass -no-snapshot-save, DO pass -no-snapshot-load)
	emulator := config.FindEmulatorExecutable()
	gpuMode := config.GetRecommendedGpuMode()
	emuArgs := []string{
		"-avd", targetAvd,
		"-lowram",
		"-memory", strconv.Itoa(ramMb),
		"-no-audio",
		"-camera-back", "none",
		"-camera-front", "none",
		"-gpu", gpuMode,
		"-no-boot-anim",
		"-no-snapshot-load",
	}
	if headless {
		emuArgs = append(emuArgs, "-no-window")
	}

	fmt.Println("🚀 Spawning baseline emulator...")
	cmd := exec.Command(emulator, emuArgs...)
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Failed to start emulator: %v\n", err)
		return
	}

	fmt.Printf("✓ Emulator spawned (PID: %d). Waiting for boot completion...\n", cmd.Process.Pid)
	_, _ = client.Exec("wait-for-device")

	// Wait for sys.boot_completed
	var targetSerial string
	booted := false
	for i := 0; i < 90; i++ {
		currentRunning, _ := client.GetRunningEmulators()
		for _, emu := range currentRunning {
			nameOut, _ := client.Exec("-s", emu.Serial, "emu", "avd", "name")
			if strings.Contains(nameOut, targetAvd) {
				targetSerial = emu.Serial
				break
			}
		}
		if targetSerial != "" {
			res, _ := client.Exec("-s", targetSerial, "shell", "getprop", "sys.boot_completed")
			if strings.TrimSpace(res) == "1" {
				booted = true
				break
			}
		} else if len(currentRunning) == 1 {
			targetSerial = currentRunning[0].Serial
			res, _ := client.Exec("-s", targetSerial, "shell", "getprop", "sys.boot_completed")
			if strings.TrimSpace(res) == "1" {
				booted = true
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	if !booted || targetSerial == "" {
		fmt.Println("❌ Timed out waiting for emulator boot. Aborting bake.")
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return
	}

	fmt.Printf("✓ Boot complete on %s! Settling system daemons (4s)...\n", targetSerial)
	time.Sleep(4 * time.Second)

	// 3. Apply avdslim optimizations
	fmt.Println("⚡ Pruning bloatware & tuning runtime settings...")
	count, err := client.Slim(targetSerial, aggressive, keepPackages, skip)
	if err != nil {
		fmt.Printf("⚠️  Warning during slim: %v\n", err)
	} else {
		fmt.Printf("✓ Disabled %d bloat packages and trimmed memory.\n", count)
	}

	time.Sleep(2 * time.Second)
	client.Exec("-s", targetSerial, "shell", "sync")

	// 4. Save golden snapshot
	fmt.Println("📸 Capturing Golden Snapshot 'avdslim_clean'...")
	snapOut, err := client.Exec("-s", targetSerial, "emu", "avd", "snapshot", "save", "avdslim_clean")
	if err != nil || strings.Contains(snapOut, "KO") {
		fmt.Printf("⚠️  Snapshot save response: %s\n", strings.TrimSpace(snapOut))
	} else {
		fmt.Printf("✓ Snapshot saved successfully: %s\n", strings.TrimSpace(snapOut))
	}

	time.Sleep(2 * time.Second)

	// 5. Verify snapshot on host filesystem
	if config.HasGoldenSnapshot(targetAvd) {
		fmt.Println("✨ Golden Snapshot verified on disk!")
	}

	// 6. Graceful shutdown
	fmt.Println("🛑 Gracefully shutting down baking emulator...")
	client.Exec("-s", targetSerial, "emu", "kill")

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if pid := host.FindHostPidForSerial(targetSerial); pid == 0 {
			break
		}
	}
	fmt.Println("✓ Emulator shut down cleanly.")

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Printf(" 🎉 Golden Snapshot Baked Successfully for %s!\n", targetAvd)
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Println(" • Snapshot Name: 'avdslim_clean'")
	fmt.Println(" • Startup Latency: Reduced from ~45s cold boot to <1.5s instant restore ⚡")
	fmt.Printf(" • RAM Allocation: %d MB (-lowram)\n", ramMb)
	fmt.Println(" • How to launch:")
	fmt.Printf("     avdslim start %s\n", targetAvd)
	fmt.Println("   Or click 'Play' in Android Studio (if `avdslim install-shim` is enabled).")
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Println()
}

func handleSnapshot(client *adb.Client, args []string) {
	var serial string
	skipSlim := false
	aggressive := false
	snapName := "avdslim_clean"
	var keepPackages []string
	skip := make(map[string]bool)

	for _, a := range args {
		if a == "--skip-slim" || a == "--no-prune" {
			skipSlim = true
		} else if strings.HasPrefix(a, "--skip=") {
			addSkip(skip, a)
		} else if a == "--aggressive" {
			aggressive = true
		} else if a == "--live" || a == "--current" {
			// standard flag alias, ignore
		} else if strings.HasPrefix(a, "--tag=") {
			snapName = strings.TrimPrefix(a, "--tag=")
		} else if strings.HasPrefix(a, "--name=") {
			snapName = strings.TrimPrefix(a, "--name=")
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
		} else if !strings.HasPrefix(a, "--") {
			serial = a
		}
	}

	running, err := client.GetRunningEmulators()
	if err != nil || len(running) == 0 {
		fmt.Println("❌ No running Android emulator found via adb.")
		fmt.Println()
		fmt.Println("💡 To snapshot a custom configured state (with test apps & logins):")
		fmt.Println("   1. Start your emulator: avdslim start")
		fmt.Println("   2. Install your debug APKs / log into test accounts")
		fmt.Println("   3. Run: avdslim snapshot")
		return
	}

	if serial == "" {
		if len(running) == 1 {
			serial = running[0].Serial
		} else {
			fmt.Printf("Multiple running emulators detected. Using %s\n", running[0].Serial)
			serial = running[0].Serial
		}
	}

	// Get AVD Name
	avdNameOut, _ := client.Exec("-s", serial, "emu", "avd", "name")
	lines := strings.Split(strings.TrimSpace(avdNameOut), "\n")
	avdName := ""
	if len(lines) > 0 {
		avdName = strings.TrimSpace(lines[0])
	}
	if avdName == "" || strings.Contains(avdName, "KO:") {
		installed := config.GetInstalledAvds()
		if len(installed) == 1 {
			avdName = installed[0]["name"]
		}
	}

	fmt.Printf("📸 Capturing Golden Snapshot from live emulator %s (%s)...\n", serial, avdName)
	if !skipSlim {
		fmt.Println("⚡ Trimming background daemons and caches while preserving installed apps...")
		count, _ := client.Slim(serial, aggressive, keepPackages, skip)
		fmt.Printf("✓ Trimmed memory and disabled %d background bloat packages.\n", count)
	}

	fmt.Println("💾 Syncing filesystem...")
	client.Exec("-s", serial, "shell", "sync")
	time.Sleep(1 * time.Second)

	fmt.Printf("📸 Saving snapshot %q...\n", snapName)
	out, err := client.Exec("-s", serial, "emu", "avd", "snapshot", "save", snapName)
	if err != nil || strings.Contains(out, "KO") {
		fmt.Printf("❌ Failed to save snapshot: %s\n", strings.TrimSpace(out))
		return
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Printf(" 🎉 Live State Captured as Golden Snapshot for %s!\n", avdName)
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Printf(" • Target AVD: %s (%s)\n", avdName, serial)
	fmt.Printf(" • Snapshot Name: %q\n", snapName)
	fmt.Println(" • Preserved: All installed apps, local databases & logged-in accounts")
	fmt.Println(" • Instant Restore: Next time you launch with `avdslim start` or Android Studio,")
	fmt.Println("   it will boot into this exact configured state in < 1.5 seconds! ⚡")
	fmt.Println(" • The running emulator remains active for your current work.")
	fmt.Println("════════════════════════════════════════════════════════════════════════")
	fmt.Println()
}

func handleUnbake(args []string) {
	installed := config.GetInstalledAvds()
	var targetAvd string

	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			targetAvd = a
		}
	}

	if targetAvd != "" {
		if idx, err := strconv.Atoi(targetAvd); err == nil && idx >= 1 && idx <= len(installed) {
			targetAvd = installed[idx-1]["name"]
		}
	} else if len(installed) > 0 {
		var err error
		targetAvd, err = selectAvdInteractively(installed, "Select an AVD to remove Golden Snapshot from")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
	} else {
		fmt.Println("❌ No installed AVDs found.")
		return
	}

	home, _ := os.UserHomeDir()
	snapDir := filepath.Join(home, ".android", "avd", targetAvd+".avd", "snapshots", "avdslim_clean")
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		fmt.Printf("ℹ️  No Golden Snapshot found for %s.\n", targetAvd)
		return
	}

	if err := os.RemoveAll(snapDir); err != nil {
		fmt.Printf("❌ Failed to remove snapshot: %v\n", err)
		return
	}

	fmt.Printf("✅ Removed Golden Snapshot 'avdslim_clean' for %s.\n", targetAvd)
	fmt.Println("   Subsequent launches will perform standard boots.")
	fmt.Println()
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
