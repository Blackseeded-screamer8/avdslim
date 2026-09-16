package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/krunalbhalala/avdslim/internal/adb"
	"github.com/krunalbhalala/avdslim/internal/config"
	"github.com/krunalbhalala/avdslim/internal/host"
)

const version = "1.0.0"

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
	case "list":
		handleList(client)
	case "measure":
		handleMeasure(client, subArgs)
	case "on", "slim":
		handleOn(client, subArgs)
	case "off", "unslim":
		handleOff(client, subArgs)
	case "tune-avd", "tune":
		handleTuneAvd(subArgs)
	case "launch":
		handleLaunch(client, subArgs)
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
  list                 List running emulators (with host RAM) & saved AVDs
  measure [device]     Deep memory breakdown (host QEMU RSS + guest dumpsys)
  on [device]          Slim down emulator: disable bloat daemons & trim RAM
                       Options: --aggressive (also disables Play Store updater)
  off [device]         Restore disabled packages and default settings
  tune-avd [avd_name]  Tune host AVD config.ini (RAM=1536M, no cameras, host GPU)
                       Options: --ram=<MB> (default: 1536), --heap=<MB> (default: 256)
  launch <avd_name>    Launch AVD with low-memory host flags (-no-audio, etc.)
                       Options: --slim (auto-slim once booted), --ram=<MB>
  version              Print avdslim version

Examples:
  avdslim list
  avdslim measure
  avdslim on
  avdslim on emulator-5554 --aggressive
  avdslim off
  avdslim tune-avd Pixel_8_API_34 --ram=1536
  avdslim launch Pixel_8_API_34 --slim
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
			hostRssMb := 0
			if hostPid > 0 {
				hostRssMb = host.GetHostRssMb(hostPid)
			}

			hostRamStr := "unknown"
			if hostRssMb > 0 {
				hostRamStr = fmt.Sprintf("%d MB", hostRssMb)
			}
			statusStr := "🔴 FULL (Stock)"
			if emu.IsSlimmed {
				statusStr = "⚡ SLIMMED"
			}
			fmt.Printf("  • %s (%s, Android %s, API %s)\n", emu.Serial, emu.Model, emu.AndroidVersion, emu.ApiLevel)
			fmt.Printf("    Host PID: %d | Host RAM (RSS): %s | Status: %s\n", hostPid, hostRamStr, statusStr)
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
		rss := host.GetHostRssMb(hostPid)
		fmt.Println("🖥️  HOST (macOS) Footprint:")
		fmt.Printf("   QEMU / Emulator PID: %d\n", hostPid)
		fmt.Printf("   Host Resident RAM (RSS): %d MB\n\n", rss)
	}

	out, _ := client.Exec("-s", serial, "shell", "dumpsys", "meminfo")
	host.PrintGuestMeminfo(out)
}

func handleOn(client *adb.Client, args []string) {
	aggressive := false
	filteredArgs := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--aggressive" {
			aggressive = true
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

	hostPid := host.FindHostPidForSerial(serial)
	beforeRss := 0
	if hostPid > 0 {
		beforeRss = host.GetHostRssMb(hostPid)
	}

	fmt.Println("1. Disabling non-essential background daemons:")
	count, _ := client.Slim(serial, aggressive)
	fmt.Printf("   -> Successfully disabled %d packages.\n\n", count)

	fmt.Println("2. Tuned system settings (animations 0x, background limit 2, sync off).")
	fmt.Println("3. Purged cached processes and trimmed memory.\n")

	time.Sleep(1 * time.Second)

	afterRss := 0
	if hostPid > 0 {
		afterRss = host.GetHostRssMb(hostPid)
	}

	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Printf("🎉 Slimming complete for %s!\n", serial)
	if beforeRss > 0 && afterRss > 0 {
		diff := beforeRss - afterRss
		if diff < 0 {
			diff = 0
		}
		fmt.Printf("🖥️  Host RAM (RSS): %dMB -> %dMB (Reclaimed: %dMB)\n", beforeRss, afterRss, diff)
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
		} else if !strings.HasPrefix(a, "--") {
			targetAvd = a
		}
	}

	if err := config.TuneAvd(targetAvd, ramMb, heapMb); err != nil {
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

	for _, a := range args[1:] {
		if a == "--slim" {
			doSlim = true
		} else if strings.HasPrefix(a, "--ram=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--ram=")); err == nil {
				ramMb = v
			}
		}
	}

	emulator := findEmulator()
	emuArgs := []string{
		"-avd", avdName,
		"-memory", strconv.Itoa(ramMb),
		"-no-audio",
		"-camera-back", "none",
		"-camera-front", "none",
		"-gpu", "host",
		"-no-boot-anim",
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

func findEmulator() string {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "Library", "Android", "sdk", "emulator", "emulator")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if p, err := exec.LookPath("emulator"); err == nil {
		return p
	}
	return "emulator"
}
