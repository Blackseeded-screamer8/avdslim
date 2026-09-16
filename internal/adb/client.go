package adb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kdbhalala/avdslim/internal/bloat"
	"github.com/kdbhalala/avdslim/internal/config"
)

const StateFilePath = "/data/local/tmp/avdslim_state.json"

type RunningEmulator struct {
	Serial         string
	Model          string
	AndroidVersion string
	ApiLevel       string
	HostPid        int
	HostRssMb      int
	IsSlimmed      bool
}

type SlimState struct {
	Timestamp        string   `json:"timestamp"`
	DisabledPackages []string `json:"disabled_packages"`
	Preset           string   `json:"preset"`
	// Settings maps "namespace/name" to its value before avdslim first changed it
	// (a nil value means it was unset). A nil map means a <= 1.0.5 state.
	Settings map[string]*string `json:"settings"`
}

// tweak is one guest setting Slim changes; Group is the name --skip accepts.
type tweak struct {
	Group, Namespace, Name, Value string
}

var tweaks = []tweak{
	{"animations", "global", "window_animation_scale", "0"},
	{"animations", "global", "transition_animation_scale", "0"},
	{"animations", "global", "animator_duration_scale", "0"},
	{"bglimit", "global", "background_process_limit", "4"},
	{"sync", "global", "auto_sync", "0"},
	{"location", "secure", "location_mode", "0"},
	{"setup", "secure", "user_setup_complete", "1"},
	{"setup", "global", "device_provisioned", "1"},
}

// SkipGroups lists the values --skip accepts.
var SkipGroups = []string{"animations", "bglimit", "sync", "location", "setup"}

// IsSkipGroup reports whether g is a valid --skip value.
func IsSkipGroup(g string) bool {
	for _, s := range SkipGroups {
		if s == g {
			return true
		}
	}
	return false
}

func (c *Client) readState(serial string) (SlimState, bool) {
	var state SlimState
	raw, err := c.Exec("-s", serial, "shell", "cat", StateFilePath)
	if err != nil || !strings.Contains(raw, "disabled_packages") {
		return state, false
	}
	return state, json.Unmarshal([]byte(raw), &state) == nil
}

func (c *Client) getSetting(serial, namespace, name string) *string {
	out, err := c.Exec("-s", serial, "shell", "settings", "get", namespace, name)
	v := strings.TrimSpace(out)
	if err != nil || v == "null" || v == "" {
		return nil
	}
	return &v
}

type Client struct {
	adbPath string
}

func NewClient() *Client {
	return &Client{
		adbPath: findAdb(),
	}
}

func (c *Client) Exec(args ...string) (string, error) {
	cmd := exec.Command(c.adbPath, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *Client) GetRunningEmulators() ([]RunningEmulator, error) {
	out, err := c.Exec("devices")
	if err != nil {
		return nil, err
	}

	var list []RunningEmulator
	scanner := bufio.NewScanner(strings.NewReader(out))
	isFirst := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isFirst {
			isFirst = false
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" && strings.HasPrefix(parts[0], "emulator-") {
			serial := parts[0]
			model, _ := c.Exec("-s", serial, "shell", "getprop", "ro.product.model")
			ver, _ := c.Exec("-s", serial, "shell", "getprop", "ro.build.version.release")
			sdk, _ := c.Exec("-s", serial, "shell", "getprop", "ro.build.version.sdk")

			stateExists, _ := c.Exec("-s", serial, "shell", "ls", StateFilePath)
			isSlimmed := strings.Contains(stateExists, "avdslim_state.json")

			list = append(list, RunningEmulator{
				Serial:         serial,
				Model:          strings.TrimSpace(model),
				AndroidVersion: strings.TrimSpace(ver),
				ApiLevel:       strings.TrimSpace(sdk),
				IsSlimmed:      isSlimmed,
			})
		}
	}
	return list, nil
}

func (c *Client) ResolveDevice(args []string) (string, error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return args[0], nil
	}
	running, err := c.GetRunningEmulators()
	if err != nil || len(running) == 0 {
		return "", fmt.Errorf("no running Android emulator found via adb")
	}
	if len(running) == 1 {
		return running[0].Serial, nil
	}
	fmt.Println("⚠️  Multiple emulators running. Please specify one:")
	for _, r := range running {
		fmt.Printf("  • %s (%s)\n", r.Serial, r.Model)
	}
	return "", fmt.Errorf("device serial required")
}

// Slim disables bloat packages and applies tweaks, except groups in skip.
func (c *Client) Slim(serial string, aggressive bool, keepPackages []string, skip map[string]bool) (int, error) {
	targetPackages := make([]string, 0, 50)
	for _, pkgs := range bloat.StandardBloatCategories {
		targetPackages = append(targetPackages, pkgs...)
	}
	if aggressive {
		targetPackages = append(targetPackages, bloat.AggressiveBloatPackages...)
	}

	keepMap := make(map[string]bool)
	for _, k := range keepPackages {
		keepMap[k] = true
	}

	installedRaw, _ := c.Exec("-s", serial, "shell", "pm", "list", "packages")
	installedMap := make(map[string]bool)
	for _, l := range strings.Split(installedRaw, "\n") {
		clean := strings.TrimSpace(strings.ReplaceAll(l, "package:", ""))
		if clean != "" {
			installedMap[clean] = true
		}
	}

	disabledList := make([]string, 0, len(targetPackages))
	for _, pkg := range targetPackages {
		if !installedMap[pkg] || keepMap[pkg] {
			continue
		}
		res, _ := c.Exec("-s", serial, "shell", "pm", "disable-user", "--user", "0", pkg)
		if strings.Contains(res, "disabled-user") || strings.Contains(res, "new state") {
			disabledList = append(disabledList, pkg)
			fmt.Printf("   ✓ Disabled: %s\n", pkg)
		}
	}

	// Persist state JSON
	preset := "Standard"
	if aggressive {
		preset = "Aggressive"
	}
	// Record each setting's original value before changing it. Keep values from an
	// earlier slim, or a re-slim would record the slimmed values as "original".
	// A <= 1.0.5 state has no originals and its values are already slimmed, so record none.
	prev, hasPrev := c.readState(serial)
	var originals map[string]*string
	if !hasPrev || prev.Settings != nil {
		originals = make(map[string]*string)
		for k, v := range prev.Settings {
			originals[k] = v
		}
		for _, t := range tweaks {
			key := t.Namespace + "/" + t.Name
			if _, ok := originals[key]; !ok && !skip[t.Group] {
				originals[key] = c.getSetting(serial, t.Namespace, t.Name)
			}
		}
	}

	state := SlimState{
		Timestamp:        time.Now().Format(time.RFC3339),
		DisabledPackages: disabledList,
		Preset:           preset,
		Settings:         originals,
	}
	stateJson, _ := json.Marshal(state)
	c.Exec("-s", serial, "shell", "echo", fmt.Sprintf("'%s'", string(stateJson)), ">", StateFilePath)

	for _, t := range tweaks {
		if !skip[t.Group] {
			c.Exec("-s", serial, "shell", "settings", "put", t.Namespace, t.Name, t.Value)
		}
	}

	// Trim memory
	c.Exec("-s", serial, "shell", "am", "kill-all")
	c.Exec("-s", serial, "shell", "am", "trim-memory", "--all", "COMPLETE")
	c.Exec("-s", serial, "shell", "su", "0", "sync")
	c.Exec("-s", serial, "shell", "su", "0", "echo 3 > /proc/sys/vm/drop_caches")

	return len(disabledList), nil
}

func (c *Client) Restore(serial string) (int, error) {
	var packagesToEnable []string
	state, hasState := c.readState(serial)
	if hasState {
		packagesToEnable = state.DisabledPackages
	}

	if len(packagesToEnable) == 0 {
		for _, pkgs := range bloat.StandardBloatCategories {
			packagesToEnable = append(packagesToEnable, pkgs...)
		}
		packagesToEnable = append(packagesToEnable, bloat.AggressiveBloatPackages...)
	}

	restoredCount := 0
	for _, pkg := range packagesToEnable {
		res, _ := c.Exec("-s", serial, "shell", "pm", "enable", pkg)
		if strings.Contains(res, "enabled") || strings.Contains(res, "new state") {
			restoredCount++
			fmt.Printf("   ✓ Enabled: %s\n", pkg)
		}
	}

	if state.Settings != nil {
		// Exact inverse: put back what Slim recorded; skipped groups were never touched.
		for _, t := range tweaks {
			orig, recorded := state.Settings[t.Namespace+"/"+t.Name]
			switch {
			case !recorded:
			case orig == nil:
				c.Exec("-s", serial, "shell", "settings", "delete", t.Namespace, t.Name)
			default:
				c.Exec("-s", serial, "shell", "settings", "put", t.Namespace, t.Name, *orig)
			}
		}
	} else {
		// ponytail: state from avdslim <= 1.0.5 has no originals, so fall back to stock values.
		c.Exec("-s", serial, "shell", "settings", "put", "global", "window_animation_scale", "1")
		c.Exec("-s", serial, "shell", "settings", "put", "global", "transition_animation_scale", "1")
		c.Exec("-s", serial, "shell", "settings", "put", "global", "animator_duration_scale", "1")
		c.Exec("-s", serial, "shell", "settings", "delete", "global", "background_process_limit")
		c.Exec("-s", serial, "shell", "settings", "put", "global", "auto_sync", "1")
		c.Exec("-s", serial, "shell", "settings", "put", "secure", "location_mode", "3")
	}
	c.Exec("-s", serial, "shell", "rm", "-f", StateFilePath)

	return restoredCount, nil
}

func findAdb() string {
	return config.FindAdbExecutable()
}
