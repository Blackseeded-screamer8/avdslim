package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/krunalbhalala/avdslim/internal/adb"
	"github.com/krunalbhalala/avdslim/internal/config"
	"github.com/krunalbhalala/avdslim/internal/host"
)

func RunDoctor(client *adb.Client) {
	fmt.Println("🩺 Running AVD-SLIM Doctor Diagnostics...\n")

	issuesCount := 0

	// 1. Toolchain Check
	fmt.Println("🛠️  1. Toolchain & Environment:")
	adbPath, err := exec.LookPath("adb")
	if err != nil {
		fmt.Println("   ❌ ADB: Not found in PATH!")
		issuesCount++
	} else {
		adbVerOut, _ := exec.Command(adbPath, "version").CombinedOutput()
		firstLine := strings.Split(string(adbVerOut), "\n")[0]
		fmt.Printf("   ✓ ADB: %s (%s)\n", adbPath, strings.TrimSpace(firstLine))
	}

	home, _ := os.UserHomeDir()
	emuPath := filepath.Join(home, "Library", "Android", "sdk", "emulator", "emulator")
	if _, err := os.Stat(emuPath); err != nil {
		if p, err := exec.LookPath("emulator"); err == nil {
			emuPath = p
		} else {
			emuPath = ""
		}
	}
	if emuPath == "" {
		fmt.Println("   ⚠️  Android Emulator: Not found in standard SDK path or PATH")
		issuesCount++
	} else {
		emuVerOut, _ := exec.Command(emuPath, "-version").CombinedOutput()
		firstLine := strings.Split(string(emuVerOut), "\n")[0]
		fmt.Printf("   ✓ Emulator: %s (%s)\n", emuPath, strings.TrimSpace(firstLine))
	}

	sdkDir := os.Getenv("ANDROID_HOME")
	if sdkDir == "" {
		sdkDir = filepath.Join(home, "Library", "Android", "sdk")
	}
	if _, err := os.Stat(sdkDir); err == nil {
		fmt.Printf("   ✓ Android SDK: %s\n\n", sdkDir)
	} else {
		fmt.Printf("   ⚠️  Android SDK: %s not found\n\n", sdkDir)
	}

	// 2. Installed AVD Configurations Audit
	fmt.Println("💾 2. Installed AVD Configurations Audit:")
	avds := config.GetInstalledAvds()
	if len(avds) == 0 {
		fmt.Println("   ℹ️  No AVDs found in ~/.android/avd\n")
	} else {
		for _, avd := range avds {
			name := avd["name"]
			ram := avd["hw.ramSize"]
			gpu := avd["hw.gpu.mode"]
			tag := avd["tag.ids"]
			target := avd["target"]

			fmt.Printf("   • AVD: %s (Target: %s)\n", name, target)

			// Check 16 KB Page Size
			if strings.Contains(tag, "page_size_16kb") || strings.Contains(avd["image.sysdir.1"], "ps16k") {
				fmt.Println("     ⚠️  16 KB Page Size image detected!")
				fmt.Println("        -> 16K images enforce 4GB minimum RAM and 4x larger page buffers.")
				fmt.Println("        -> Tip: For daily Flutter/Android dev, standard 4K 'Google APIs' images use 60% less RAM.")
				issuesCount++
			} else {
				fmt.Println("     ✓ Standard 4 KB memory pages (optimal)")
			}

			// Check RAM allocation
			if ram == "1536" || ram == "1024" || ram == "1280" {
				fmt.Printf("     ✓ Tuned low-RAM allocation: %s MB\n", ram)
			} else {
				fmt.Printf("     ⚠️  High RAM allocated: %s MB (Recommend: 1536 MB via `avdslim tune-avd %s`)\n", ram, name)
				issuesCount++
			}

			// Check GPU Mode
			if gpu == "host" {
				fmt.Println("     ✓ Metal Hardware GPU Acceleration enabled (`host`)")
			} else {
				fmt.Printf("     ⚠️  GPU Mode is %q (Recommend: `host` for native Apple Silicon Metal GPU)\n", gpu)
				issuesCount++
			}
		}
		fmt.Println()
	}

	// 3. Running Emulators Diagnostic
	fmt.Println("📱 3. Running Emulator Diagnostics:")
	running, _ := client.GetRunningEmulators()
	if len(running) == 0 {
		fmt.Println("   ℹ️  No active emulators running right now.\n")
	} else {
		for _, emu := range running {
			hostPid := host.FindHostPidForSerial(emu.Serial)
			fp := 0
			rss := 0
			if hostPid > 0 {
				fp = host.GetHostFootprintMb(hostPid)
				rss = host.GetHostRssMb(hostPid)
			}

			status := "🔴 Stock (High Memory)"
			if emu.IsSlimmed {
				status = "⚡ Slimmed"
			}

			fmt.Printf("   • %s (%s, API %s)\n", emu.Serial, emu.Model, emu.ApiLevel)
			fmt.Printf("     - Status: %s\n", status)
			if hostPid > 0 {
				fmt.Printf("     - Host PID: %d\n", hostPid)
				fmt.Printf("     - Activity Monitor Footprint: %d MB\n", fp)
				fmt.Printf("     - Physical RSS: %d MB\n", rss)
			}

			// Check guest root capability
			rootCheck, _ := client.Exec("-s", emu.Serial, "shell", "su", "0", "id")
			if strings.Contains(rootCheck, "uid=0") {
				fmt.Println("     ✓ Guest Root (`su 0`): Available (Enables kernel pagecache trimming)")
			} else {
				fmt.Println("     ℹ️  Guest Root: Not available (Google Play production build)")
			}

			// Check guest RAM
			meminfo, _ := client.Exec("-s", emu.Serial, "shell", "dumpsys", "meminfo")
			for _, line := range strings.Split(meminfo, "\n") {
				if strings.Contains(line, "Total RAM:") {
					fmt.Printf("     - Guest OS RAM: %s\n", strings.TrimSpace(line))
				}
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("══════════════════════════════════════════════════════════════")
	if issuesCount == 0 {
		fmt.Println("🎉 Everything looks optimal! Emulators are tuned for maximum efficiency.")
	} else {
		fmt.Printf("ℹ️  Doctor found %d recommendation(s) to reduce memory overhead.\n", issuesCount)
		fmt.Println("   Run `avdslim tune-avd <name>` and `avdslim restart` to apply recommendations.")
	}
	fmt.Println("══════════════════════════════════════════════════════════════\n")
}
