package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func GetInstalledAvds() []map[string]string {
	avdBase := GetAvdBaseDir()

	var list []map[string]string
	entries, err := os.ReadDir(avdBase)
	if err != nil {
		return list
	}

	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".avd") {
			cfgPath := filepath.Join(avdBase, e.Name(), "config.ini")
			data, err := os.ReadFile(cfgPath)
			if err != nil {
				continue
			}
			m := map[string]string{
				"name": strings.TrimSuffix(e.Name(), ".avd"),
			}
			for _, line := range strings.Split(string(data), "\n") {
				idx := strings.Index(line, "=")
				if idx > 0 {
					k := strings.TrimSpace(line[:idx])
					v := strings.TrimSpace(line[idx+1:])
					m[k] = v
				}
			}
			list = append(list, m)
		}
	}
	return list
}

func TuneAvd(targetAvd string, ramMb, heapMb int, gpuMode string) error {
	avdBase := GetAvdBaseDir()

	var configFiles []string
	filepath.Walk(avdBase, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.Name() == "config.ini" {
			configFiles = append(configFiles, path)
		}
		return nil
	})

	if len(configFiles) == 0 {
		return fmt.Errorf("no AVD configurations found in %s", avdBase)
	}

	var fileToTune string
	if targetAvd != "" {
		for _, f := range configFiles {
			avdName := strings.TrimSuffix(filepath.Base(filepath.Dir(f)), ".avd")
			if strings.EqualFold(avdName, targetAvd) {
				fileToTune = f
				break
			}
		}
		if fileToTune == "" {
			return fmt.Errorf("AVD %q not found in ~/.android/avd", targetAvd)
		}
	} else if len(configFiles) == 1 {
		fileToTune = configFiles[0]
	} else {
		fmt.Println("Found multiple AVDs. Please specify one:")
		for _, f := range configFiles {
			fmt.Printf("  • %s\n", strings.TrimSuffix(filepath.Base(filepath.Dir(f)), ".avd"))
		}
		return fmt.Errorf("please specify AVD name")
	}

	avdName := strings.TrimSuffix(filepath.Base(filepath.Dir(fileToTune)), ".avd")
	fmt.Printf("⚙️  Tuning config.ini for %q (RAM: %dMB, Heap: %dMB)...\n", avdName, ramMb, heapMb)

	data, err := os.ReadFile(fileToTune)
	if err != nil {
		return err
	}

	backupFile := fileToTune + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		_ = os.WriteFile(backupFile, data, 0644)
		fmt.Printf("   ✓ Created backup: %s\n", backupFile)
	}

	lines := strings.Split(string(data), "\n")
	kv := make(map[string]string)
	for _, l := range lines {
		idx := strings.Index(l, "=")
		if idx > 0 {
			k := strings.TrimSpace(l[:idx])
			v := strings.TrimSpace(l[idx+1:])
			kv[k] = v
		}
	}

	if gpuMode == "" {
		gpuMode = GetRecommendedGpuMode()
	}

	kv["hw.ramSize"] = strconv.Itoa(ramMb)
	kv["vm.heapSize"] = strconv.Itoa(heapMb)
	kv["hw.camera.back"] = "none"
	kv["hw.camera.front"] = "none"
	kv["hw.audioInput"] = "no"
	kv["hw.audioOutput"] = "no"
	kv["hw.gpu.mode"] = gpuMode
	kv["hw.gpu.enabled"] = "yes"
	kv["hw.keyboard"] = "yes"
	kv["hw.cpu.ncore"] = "2"
	kv["hw.dPad"] = "no"
	kv["fastboot.forceColdBoot"] = "yes"
	kv["fastboot.forceFastBoot"] = "no"

	var sb strings.Builder
	for k, v := range kv {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	if err := os.WriteFile(fileToTune, []byte(sb.String()), 0644); err != nil {
		return err
	}

	// Purge stale runtime ini and snapshots to prevent restoring previous 4GB/lavapipe states
	avdDir := filepath.Dir(fileToTune)
	_ = os.Remove(filepath.Join(avdDir, "hardware-qemu.ini"))
	_ = os.Remove(filepath.Join(avdDir, "hardware-qemu.ini.lock"))
	_ = os.RemoveAll(filepath.Join(avdDir, "snapshots"))

	fmt.Printf("✅ Successfully tuned AVD %q!\n", avdName)
	fmt.Printf("   • Host RAM allocated: %d MB (prevents host memory pressure)\n", ramMb)
	fmt.Printf("   • VM Heap: %d MB\n", heapMb)
	fmt.Println("   • Hardware Audio & Camera: disabled (saves host threads/buffers)")
	fmt.Printf("   • GPU Mode: %s (%s)\n", gpuMode, GetGpuBackendDescription(gpuMode))
	fmt.Println("   • Runtime cache & snapshots: purged (prevents restoring stale 4GB/lavapipe states)")

	is16K := strings.Contains(kv["tag.id"], "page_size_16kb") || strings.Contains(kv["image.sysdir.1"], "ps16k") || strings.Contains(kv["image.sysdir.1"], "16kb")
	isPlayStore := strings.ToLower(kv["PlayStore.enabled"]) == "true" || strings.ToLower(kv["PlayStore.enabled"]) == "yes" || strings.Contains(kv["tag.id"], "playstore")
	if is16K {
		fmt.Println("   🚨 Note: This AVD uses a 16 KB page-size image. QEMU enforces a 4096 MB RAM floor.")
		fmt.Println("      💡 Recommendation: For daily dev at 1536 MB RAM, use standard 4 KB 'Google APIs'.")
	} else if isPlayStore {
		fmt.Println("   ⚠️  Note: This AVD uses 'Google Play' (locks guest root & runs background updaters).")
		fmt.Println("      💡 Recommendation: For lowest RAM usage, use 'Google APIs' instead.")
	}

	fmt.Println()
	fmt.Printf("ℹ️  Note: If this emulator is currently running, restart it to apply changes:\n   avdslim restart %s\n\n", avdName)
	return nil
}

func HasGoldenSnapshot(avdName string) bool {
	avdBase := GetAvdBaseDir()
	snapDir := filepath.Join(avdBase, avdName+".avd", "snapshots", "avdslim_clean")
	info, err := os.Stat(snapDir)
	return err == nil && info.IsDir()
}
