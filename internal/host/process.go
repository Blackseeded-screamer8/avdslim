package host

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func FindHostPidForSerial(serial string) int {
	portStr := strings.TrimPrefix(serial, "emulator-")

	// 1. Try lsof on console port (e.g. 5554) - fastest and most precise
	if portStr != "" {
		cmd := exec.Command("lsof", "-i", ":"+portStr, "-sTCP:LISTEN", "-t")
		if out, err := cmd.CombinedOutput(); err == nil {
			fields := strings.Fields(string(out))
			if len(fields) > 0 {
				if pid, err := strconv.Atoi(fields[0]); err == nil && pid > 0 {
					return pid
				}
			}
		}
	}

	// 2. Fallback to process inspection
	cmd := exec.Command("ps", "-eo", "pid,command")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	myPid := os.Getpid()
	lines := strings.Split(string(out), "\n")
	var candidates []int

	for _, l := range lines {
		if strings.Contains(l, "avdslim") || strings.Contains(l, "crashpad") || strings.Contains(l, "netsimd") {
			continue
		}

		if strings.Contains(l, "qemu-system") {
			fields := strings.Fields(l)
			if len(fields) > 0 {
				if pid, err := strconv.Atoi(fields[0]); err == nil && pid != myPid {
					if strings.Contains(l, "-port "+portStr) || strings.Contains(l, portStr) {
						return pid
					}
					candidates = append(candidates, pid)
				}
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return 0
}

func GetHostRssMb(pid int) int {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.Atoi(fields[1]); err == nil {
							return kb / 1024
						}
					}
				}
			}
		}
	}

	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "rss=")
	out, err := cmd.CombinedOutput()
	if err == nil {
		if kb, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			return kb / 1024
		}
	}
	return 0
}

func parseFootprintOutput(out string) int {
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, "phys_footprint:")
		if idx == -1 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len("phys_footprint:"):])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}

		valStr := fields[0]
		unit := ""
		if len(fields) >= 2 {
			unit = strings.ToUpper(fields[1])
		} else {
			for i, ch := range valStr {
				if (ch < '0' || ch > '9') && ch != '.' {
					unit = strings.ToUpper(valStr[i:])
					valStr = valStr[:i]
					break
				}
			}
		}

		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil || val <= 0 {
			continue
		}

		switch unit {
		case "B", "BYTE", "BYTES":
			return int(val / (1024 * 1024))
		case "KB", "K", "KIB":
			return int(val / 1024)
		case "MB", "M", "MIB":
			return int(val)
		case "GB", "G", "GIB":
			return int(val * 1024)
		case "":
			if val > 1048576 {
				return int(val / (1024 * 1024))
			}
			return int(val)
		default:
			return int(val)
		}
	}
	return 0
}

func GetHostFootprintMb(pid int) int {
	if runtime.GOOS == "darwin" {
		// 1. Try footprint with --format bytes for exact byte-level precision
		cmdBytes := exec.Command("footprint", "--format", "bytes", "-p", strconv.Itoa(pid))
		if out, err := cmdBytes.CombinedOutput(); err == nil {
			if mb := parseFootprintOutput(string(out)); mb > 0 {
				return mb
			}
		}

		// 2. Fallback to standard footprint output with unit parsing
		cmd := exec.Command("footprint", "-p", strconv.Itoa(pid))
		if out, err := cmd.CombinedOutput(); err == nil {
			if mb := parseFootprintOutput(string(out)); mb > 0 {
				return mb
			}
		}
	}
	return GetHostRssMb(pid)
}

func PrintGuestMeminfo(raw string) {
	fmt.Println("📱 GUEST (Android OS) Memory Breakdown:")
	lines := strings.Split(raw, "\n")
	for _, l := range lines {
		if strings.Contains(l, "Total RAM:") || strings.Contains(l, "Free RAM:") ||
			strings.Contains(l, "Used RAM:") || strings.Contains(l, "Lost RAM:") {
			fmt.Printf("   %s\n", strings.TrimSpace(l))
		}
	}
	fmt.Println()

	fmt.Println("🔝 Top Memory-Consuming Processes in Guest:")
	found := false
	count := 0
	for _, l := range lines {
		if strings.Contains(l, "Total PSS by process:") {
			found = true
			continue
		}
		if found {
			t := strings.TrimSpace(l)
			if t == "" || strings.HasPrefix(t, "Total PSS by category:") {
				break
			}
			if strings.Contains(t, "K:") && count < 10 {
				fmt.Printf("   %s\n", t)
				count++
			}
		}
	}
	fmt.Println()
}
