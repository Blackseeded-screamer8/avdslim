package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/krunalbhalala/avdslim/internal/adb"
	"github.com/krunalbhalala/avdslim/internal/bloat"
	"github.com/krunalbhalala/avdslim/internal/config"
	"github.com/krunalbhalala/avdslim/internal/doctor"
	"github.com/krunalbhalala/avdslim/internal/host"
)

const version = "1.0.1"

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
	case "launch":
		handleLaunch(client, subArgs)
	case "restart":
		handleRestart(client, subArgs)
	default:
		fmt.Printf("❌ Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
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
  restore, off         Instant 100%% stock restore (re-enables packages, animations & sync)
  watch                Auto-detect & slim new emulators as soon as they boot
                       Options: --aggressive, --keep=<package>
  tune-avd [avd_name]  Tune host AVD config.ini (RAM=1536M, Metal GPU, no cameras)
                       Options: --ram=<MB> (default: 1536), --heap=<MB> (default: 256)
  launch <avd_name>    Launch AVD with low-memory host flags (-lowram, -memory, etc.)
                       Options: --slim (auto-slim once booted), --ram=<MB>
  restart [device]     Gracefully restart emulator with clean cache, low-memory flags
                       Options: --ram=<MB> (default: 1536)
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
  avdslim tune-avd Pixel_10_Pro --ram=1536
  avdslim restart
`, version)
}

func handleList(client *adb.Client) {
	fmt.Println("🔎 Checking running Android emulators...")
	running, err := client.GetRunningEmulators()
	if err != nil {
		fmt.Printf("Error checking emulators: %v\n", err)
	} else if len(running) == 0 {
		fmt.Println("ℹ️  No running Android emulators detected via adb.\n")
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
		fmt.Println("  (No AVDs found in ~/.android/avd)\n")
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
	for _, a := range args {
		if a == "--aggressive" {
			aggressive = true
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
		} else {
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
	count, _ := client.Slim(serial, aggressive, keepPackages)
	fmt.Printf("   -> Successfully disabled %d packages.\n\n", count)

	fmt.Println("2. Tuned system settings (animations 0x, background limit 2, sync off).")
	fmt.Println("3. Purged cached processes and trimmed memory.\n")

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
	fmt.Println("2. Restored default system settings (animations 1.0x, auto-sync on).")
	fmt.Printf("✅ Successfully restored %s to stock configuration.\n\n", serial)
}

func handleTuneAvd(args []string) {
	ramMb := 1536
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

	if err := config.TuneAvd(targetAvd, ramMb, heapMb, gpuMode); err != nil {
		fmt.Printf("❌ %v\n", err)
	}
}

func handleLaunch(client *adb.Client, args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		fmt.Println("❌ Please specify an AVD name to launch.")
		fmt.Println("   Example: avdslim launch Pixel_8_API_34 --slim")
		return
	}

	avdName := args[0]
	doSlim := false
	ramMb := 1536
	gpuMode := config.GetRecommendedGpuMode()

	for _, a := range args[1:] {
		if a == "--slim" {
			doSlim = true
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
		"-lowram",
		"-memory", strconv.Itoa(ramMb),
		"-no-audio",
		"-camera-back", "none",
		"-camera-front", "none",
		"-gpu", gpuMode,
		"-no-boot-anim",
		"-no-snapshot-load",
	}

	fmt.Printf("🚀 Launching emulator %q with low-memory host flags:\n", avdName)
	fmt.Printf("   emulator %s\n\n", strings.Join(emuArgs, " "))

	cmd := exec.Command(emulator, emuArgs...)
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
			fmt.Println("✓ Boot complete! Applying avdslim optimizations...")
			handleOn(client, nil)
		} else {
			fmt.Println("⚠️  Boot timed out after 120s. You can run `avdslim on` manually.")
		}
	}
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
	for _, a := range args {
		if a == "--aggressive" {
			aggressive = true
		} else if strings.HasPrefix(a, "--keep=") {
			pkg := strings.TrimPrefix(a, "--keep=")
			if pkg != "" {
				keepPackages = append(keepPackages, pkg)
			}
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
	fmt.Println("   Monitoring for newly booted Android emulators in the background.")
	fmt.Println("   Will automatically apply low-memory optimizations as soon as emulators boot.")
	fmt.Println("   Press Ctrl+C to stop.\n")

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
				count, _ := client.Slim(emu.Serial, aggressive, keepPackages)
				fmt.Printf("✓ [%s] Successfully slimmed! Disabled %d packages, trimmed RAM.\n\n", emu.Serial, count)
				slimmedDevices[emu.Serial] = true
			}
		}
	}
}

