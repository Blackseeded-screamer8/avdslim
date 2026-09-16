package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func GetInstalledAvds() []map[string]string {
	homeDir, _ := os.UserHomeDir()
	avdBase := filepath.Join(homeDir, ".android", "avd")

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

func TuneAvd(targetAvd string, ramMb, heapMb int) error {
	homeDir, _ := os.UserHomeDir()
	avdBase := filepath.Join(homeDir, ".android", "avd")

	var configFiles []string
	filepath.Walk(avdBase, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.Name() == "config.ini" {
			configFiles = append(configFiles, path)
		}
		return nil
	})

	if len(configFiles) == 0 {
		return fmt.Errorf("no AVD configurations found in ~/.android/avd")
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

	kv["hw.ramSize"] = strconv.Itoa(ramMb)
	kv["vm.heapSize"] = strconv.Itoa(heapMb)
	kv["hw.camera.back"] = "none"
	kv["hw.camera.front"] = "none"
	kv["hw.audioInput"] = "no"
	kv["hw.audioOutput"] = "no"
	kv["hw.gpu.mode"] = "host"
	kv["hw.gpu.enabled"] = "yes"
	kv["hw.dPad"] = "no"

	var sb strings.Builder
	for k, v := range kv {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	if err := os.WriteFile(fileToTune, []byte(sb.String()), 0644); err != nil {
		return err
	}

	fmt.Printf("✅ Successfully tuned AVD %q!\n", avdName)
	fmt.Printf("   • Host RAM allocated: %d MB (prevents host memory pressure)\n", ramMb)
	fmt.Printf("   • VM Heap: %d MB\n", heapMb)
	fmt.Println("   • Hardware Audio & Camera: disabled (saves host threads/buffers)")
	fmt.Println("   • GPU Mode: host (uses Apple Silicon Metal hardware acceleration)\n")
	return nil
}
