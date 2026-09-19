package host

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// HostMemoryInfo contains host physical RAM and swap metrics.
type HostMemoryInfo struct {
	TotalMb        int
	AvailableMb    int
	FreeMb         int
	SwapTotalMb    int
	SwapUsedMb     int
	PageSizeBytes  int
	IsAppleSilicon bool
}

// GetHostMemoryInfo inspects the host machine's RAM and swap usage.
// On macOS, it derives page size via os.Getpagesize() to properly handle Apple Silicon (16 KB)
// vs Intel (4 KB) and parses vm_stat and sysctl vm.swapusage.
// On Linux, it inspects /proc/meminfo.
func GetHostMemoryInfo() HostMemoryInfo {
	info := HostMemoryInfo{
		PageSizeBytes: os.Getpagesize(),
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		info.IsAppleSilicon = true
	}

	if runtime.GOOS == "darwin" {
		populateDarwinMemoryInfo(&info)
	} else if runtime.GOOS == "linux" {
		populateLinuxMemoryInfo(&info)
	}

	return info
}

func populateDarwinMemoryInfo(info *HostMemoryInfo) {
	// 1. Total Physical RAM via sysctl hw.memsize
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").CombinedOutput(); err == nil {
		if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil && bytes > 0 {
			info.TotalMb = int(bytes / (1024 * 1024))
		}
	}

	// 2. Free & Available RAM via vm_stat with exact page size
	if out, err := exec.Command("vm_stat").CombinedOutput(); err == nil {
		freeMb, availMb := parseVmStatOutput(string(out), info.PageSizeBytes)
		info.FreeMb = freeMb
		info.AvailableMb = availMb
	}

	// 3. Swap usage via sysctl vm.swapusage
	if out, err := exec.Command("sysctl", "-n", "vm.swapusage").CombinedOutput(); err == nil {
		totalMb, usedMb := parseSwapUsageDarwin(string(out))
		info.SwapTotalMb = totalMb
		info.SwapUsedMb = usedMb
	}
}

// parseVmStatOutput calculates free and available memory from vm_stat output using the system page size.
func parseVmStatOutput(out string, pageSizeBytes int) (int, int) {
	if pageSizeBytes <= 0 {
		pageSizeBytes = 4096
	}
	pageSize := int64(pageSizeBytes)

	var pagesFree, pagesSpeculative, pagesInactive, pagesPurgeable int64

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.Trim(strings.TrimSpace(parts[1]), ".")
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "Pages free":
			pagesFree = val
		case "Pages speculative":
			pagesSpeculative = val
		case "Pages inactive":
			pagesInactive = val
		case "Pages purgeable":
			pagesPurgeable = val
		}
	}

	freeMb := int((pagesFree + pagesSpeculative) * pageSize / (1024 * 1024))
	availMb := int((pagesFree + pagesSpeculative + pagesInactive + pagesPurgeable) * pageSize / (1024 * 1024))
	return freeMb, availMb
}

// parseSwapUsageDarwin parses strings like:
// "total = 6144.00M  used = 5748.69M  free = 395.31M  (encrypted)"
func parseSwapUsageDarwin(s string) (int, int) {
	fields := strings.Fields(s)
	totalMb := 0
	usedMb := 0
	for i := 0; i < len(fields)-2; i++ {
		if fields[i] == "total" && fields[i+1] == "=" {
			totalMb = parseSizeMb(fields[i+2])
		}
		if fields[i] == "used" && fields[i+1] == "=" {
			usedMb = parseSizeMb(fields[i+2])
		}
	}
	return totalMb, usedMb
}

// parseSizeMb parses strings like "6144.00M", "1.50G", "512K", or "1048576" into megabytes.
func parseSizeMb(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	unit := ""
	valStr := s
	for i, ch := range s {
		if (ch < '0' || ch > '9') && ch != '.' {
			unit = strings.ToUpper(s[i:])
			valStr = s[:i]
			break
		}
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil || val <= 0 {
		return 0
	}

	switch unit {
	case "G", "GB", "GIB":
		return int(val * 1024)
	case "M", "MB", "MIB", "":
		return int(val)
	case "K", "KB", "KIB":
		return int(val / 1024)
	case "B", "BYTE", "BYTES":
		return int(val / (1024 * 1024))
	default:
		return int(val)
	}
}

func populateLinuxMemoryInfo(info *HostMemoryInfo) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}
	totalMb, freeMb, availMb, swapTotalMb, swapUsedMb := parseLinuxMemInfo(data)
	info.TotalMb = totalMb
	info.FreeMb = freeMb
	info.AvailableMb = availMb
	info.SwapTotalMb = swapTotalMb
	info.SwapUsedMb = swapUsedMb
}

func parseLinuxMemInfo(data []byte) (int, int, int, int, int) {
	var totalMb, freeMb, availMb, swapTotalMb, swapUsedMb int
	var swapTotalKb, swapFreeKb int

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalMb = val / 1024
		case "MemFree:":
			freeMb = val / 1024
		case "MemAvailable:":
			availMb = val / 1024
		case "SwapTotal:":
			swapTotalKb = val
			swapTotalMb = val / 1024
		case "SwapFree:":
			swapFreeKb = val
		}
	}

	if swapTotalKb > 0 {
		swapUsedMb = (swapTotalKb - swapFreeKb) / 1024
		if swapUsedMb < 0 {
			swapUsedMb = 0
		}
	}
	if availMb == 0 && freeMb > 0 {
		availMb = freeMb
	}

	return totalMb, freeMb, availMb, swapTotalMb, swapUsedMb
}
