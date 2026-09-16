package doctor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/krunalbhalala/avdslim/internal/adb"
	"github.com/krunalbhalala/avdslim/internal/config"
	"github.com/krunalbhalala/avdslim/internal/host"
	"github.com/krunalbhalala/avdslim/internal/shim"
)

func RunDoctor(client *adb.Client) {
	fmt.Println("🩺 Running AVD-SLIM Doctor Diagnostics...")
	fmt.Println()

	issuesCount := 0

	// 1. Toolchain Check
	fmt.Println("🛠️  1. Toolchain & Environment:")
	adbPath := config.FindAdbExecutable()
	if adbPath == "adb" {
		if _, err := exec.LookPath("adb"); err != nil {
			fmt.Println("   ❌ ADB: Not found in standard SDK path or PATH!")
			issuesCount++
		} else {
			adbVerOut, _ := exec.Command(adbPath, "version").CombinedOutput()
			firstLine := strings.Split(string(adbVerOut), "\n")[0]
			fmt.Printf("   ✓ ADB: %s (%s)\n", adbPath, strings.TrimSpace(firstLine))
		}
	} else {
		adbVerOut, _ := exec.Command(adbPath, "version").CombinedOutput()
		firstLine := strings.Split(string(adbVerOut), "\n")[0]
		fmt.Printf("   ✓ ADB: %s (%s)\n", adbPath, strings.TrimSpace(firstLine))
	}

	emuPath := config.FindEmulatorExecutable()
	if emuPath == "emulator" {
		if _, err := exec.LookPath("emulator"); err != nil {
			fmt.Println("   ⚠️  Android Emulator: Not found in standard SDK path or PATH")
			issuesCount++
		} else {
			emuVerOut, _ := exec.Command(emuPath, "-version").CombinedOutput()
			firstLine := strings.Split(string(emuVerOut), "\n")[0]
			fmt.Printf("   ✓ Emulator: %s (%s)\n", emuPath, strings.TrimSpace(firstLine))
		}
	} else {
		emuVerOut, _ := exec.Command(emuPath, "-version").CombinedOutput()
		firstLine := strings.Split(string(emuVerOut), "\n")[0]
		fmt.Printf("   ✓ Emulator: %s (%s)\n", emuPath, strings.TrimSpace(firstLine))
	}

	shimActive, _ := shim.IsShimInstalled()
	if shimActive {
		fmt.Println("   ✓ Android Studio Shim: Active (GUI launches are automatically slimmed)")
	} else {
		fmt.Println("   ℹ️  Android Studio Shim: Not installed (run `avdslim install-shim` to auto-slim Studio launches)")
	}

	sdkDir := config.GetAndroidSdkDir()
	if sdkDir != "" {
		fmt.Printf("   ✓ Android SDK: %s\n\n", sdkDir)
	} else {
		fmt.Println("   ⚠️  Android SDK directory not found ($ANDROID_HOME not set)")
		fmt.Println()
	}

	// 2. Installed AVD Configurations Audit
	fmt.Println("💾 2. Installed AVD Configurations Audit:")
	avds := config.GetInstalledAvds()
	if len(avds) == 0 {
		fmt.Printf("   ℹ️  No AVDs found in %s\n\n", config.GetAvdBaseDir())
	} else {
		for _, avd := range avds {
			name := avd["name"]
			ram := avd["hw.ramSize"]
			gpu := avd["hw.gpu.mode"]
			tag := avd["tag.ids"]
			target := avd["target"]

			fmt.Printf("   • AVD: %s (Target: %s)\n", name, target)

			// Check System Image Type & Recommendation
			is16K := strings.Contains(tag, "page_size_16kb") || strings.Contains(avd["image.sysdir.1"], "ps16k") || strings.Contains(avd["image.sysdir.1"], "16kb")
			isPlayStore := strings.ToLower(avd["PlayStore.enabled"]) == "true" || strings.ToLower(avd["PlayStore.enabled"]) == "yes" || strings.Contains(tag, "playstore") || strings.Contains(avd["tag.id"], "playstore")

			if is16K {
				fmt.Println("     🚨 [AVOID] 16 KB Page Size Image Detected!")
				fmt.Println("        -> QEMU enforces 4GB minimum RAM ceiling for 16K images, ignoring low-RAM flags.")
				fmt.Println("        -> Allocates 4x larger page buffers in host memory.")
				fmt.Println("        -> 💡 Recommendation: In SDK Manager, switch to standard 'Google APIs' (4 KB) image.")
				issuesCount++
			} else if isPlayStore {
				fmt.Println("     ⚠️  [AVOID] 'Google Play' Production Image Detected")
				fmt.Println("        -> Runs heavy Play Store auto-updater daemons and background security scans.")
				fmt.Println("        -> Production build locks out `adb root` (cannot drop Linux kernel dirty pagecaches).")
				fmt.Println("        -> 💡 Recommendation: Use 'Google APIs' image instead (100% Firebase/FCM, 40% less RAM, adb root enabled).")
				issuesCount++
			} else {
				fmt.Println("     ✅ [OPTIMAL] 'Google APIs' (Standard 4 KB pages, Firebase/FCM enabled, adb root capable)")
			}

			// Check RAM allocation
			if ram == "1024" || ram == "1536" || ram == "1280" {
				fmt.Printf("     ✓ Tuned low-RAM allocation: %s MB\n", ram)
			} else {
				fmt.Printf("     ⚠️  High RAM allocated: %s MB (Recommend: 1024 MB via `avdslim tune-avd %s`)\n", ram, name)
				issuesCount++
			}

			// Check GPU Mode
			if gpu == "host" || gpu == "swiftshader_indirect" || gpu == "angle_indirect" {
				fmt.Printf("     ✓ Hardware GPU acceleration enabled: %s (%s)\n", gpu, config.GetGpuBackendDescription(gpu))
			} else {
				fmt.Printf("     ⚠️  GPU Mode is %q (Recommend: `host` for native hardware acceleration)\n", gpu)
				issuesCount++
			}

			// Check Golden Snapshot
			if config.HasGoldenSnapshot(name) {
				fmt.Println("     ✨ Golden Snapshot: Baked ('avdslim_clean' ~1.5s instant boot ready)")
			} else {
				fmt.Printf("     ℹ️  Golden Snapshot: None (Run `avdslim bake %s` for ~1.5s instant boot)\n", name)
			}
		}
		fmt.Println()
	}

	// 3. Running Emulators Diagnostic
	fmt.Println("📱 3. Running Emulator Diagnostics:")
	running, _ := client.GetRunningEmulators()
	if len(running) == 0 {
		fmt.Println("   ℹ️  No active emulators running right now.")
		fmt.Println()
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

	// 4. Golden SDK Recommendation Banner
	fmt.Println("💡 4. Golden SDK System Image Recommendation:")
	fmt.Println("┌────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  When creating AVDs in Android Studio Device Manager:                  │")
	fmt.Println("│                                                                        │")
	fmt.Println("│  ✅ ALWAYS CHOOSE: \"Google APIs\" (Standard 4 KB pages)                 │")
	fmt.Println("│     • 100% Firebase Auth, FCM Push, Google Sign-In & Maps support       │")
	fmt.Println("│     • Guest root (`adb root`) enabled for instant kernel cache drops   │")
	fmt.Println("│     • Runs smoothly with 1024 MB RAM (saves 60-70% host memory)        │")
	fmt.Println("│                                                                        │")
	fmt.Println("│  ❌ AVOID: \"Google Play\"                                               │")
	fmt.Println("│     • Adds heavy Play Store self-updaters & Play Protect scanning loops│")
	fmt.Println("│     • Production build locks out `adb root`                            │")
	fmt.Println("│                                                                        │")
	fmt.Println("│  ❌ AVOID: \"16 KB Page Size\" (ps16k)                                   │")
	fmt.Println("│     • Hardcodes 4096 MB minimum RAM ceiling in QEMU                    │")
	fmt.Println("│     • 4x larger page buffers consume 2.5x more host memory              │")
	fmt.Println("└────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Summary
	fmt.Println("══════════════════════════════════════════════════════════════")
	if issuesCount == 0 {
		fmt.Println("🎉 Everything looks optimal! Emulators are tuned for maximum efficiency.")
	} else {
		fmt.Printf("ℹ️  Doctor found %d recommendation(s) to reduce memory overhead.\n", issuesCount)
		fmt.Println("   Run `avdslim tune-avd <name>` and `avdslim restart` to apply recommendations.")
	}
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println()
}
