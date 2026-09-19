package host

import (
	"os"
	"testing"
)

func TestParseVmStatOutputAppleSilicon(t *testing.T) {
	// Sample vm_stat on Apple Silicon (16 KB page size)
	// Using 6400 free pages so division by 1048576 is exact:
	// 6400 * 16384 = 104,857,600 bytes = exactly 100 MB.
	// 6400 * 4096 = 26,214,400 bytes = exactly 25 MB.
	vmStatOutput := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                     6000.
Pages active:                                 100000.
Pages inactive:                                50000.
Pages speculative:                               400.
Pages throttled:                                   0.
Pages wired down:                              80000.
Pages purgeable:                                2000.
"Translation faults":                     1234567890.
Pages copy-on-write:                       123456789.
Pages zero filled:                         123456789.
Pages reactivated:                          12345678.
Pages purged:                                1234567.
File-backed pages:                             50000.
Anonymous pages:                               50000.
Pages stored in compressor:                    50000.
Pages occupied by compressor:                  20000.
`
	pageSize16K := 16384 // 16 KB

	freeMb, availMb := parseVmStatOutput(vmStatOutput, pageSize16K)

	// Free = (6000 + 400) * 16384 / 1048576 = 6400 * 16 / 1024 = 100 MB
	expectedFreeMb := 100
	if freeMb != expectedFreeMb {
		t.Errorf("freeMb = %d, expected %d", freeMb, expectedFreeMb)
	}

	// Avail = (6000 + 400 + 50000 + 2000) * 16384 / 1048576 = 58400 * 16 / 1024 = 912.5 -> 912 MB
	expectedAvailMb := int(int64(58400) * 16384 / (1024 * 1024))
	if availMb != expectedAvailMb {
		t.Errorf("availMb = %d, expected %d", availMb, expectedAvailMb)
	}

	// Verify that assuming 4096 (x86) would be 4x off
	free4k, _ := parseVmStatOutput(vmStatOutput, 4096)
	if free4k*4 != freeMb {
		t.Errorf("expected 4x difference between 16KB and 4KB page size, got free4k=%d vs freeMb=%d", free4k, freeMb)
	}
}

func TestParseSwapUsageDarwin(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTotal int
		expectedUsed  int
	}{
		{
			name:          "standard macOS swapusage output in megabytes",
			input:         "total = 6144.00M  used = 5748.69M  free = 395.31M  (encrypted)",
			expectedTotal: 6144,
			expectedUsed:  5748,
		},
		{
			name:          "gigabytes format",
			input:         "total = 4.00G  used = 1.50G  free = 2.50G",
			expectedTotal: 4096,
			expectedUsed:  1536,
		},
		{
			name:          "zero swap used",
			input:         "total = 1024.00M  used = 0.00M  free = 1024.00M  (encrypted)",
			expectedTotal: 1024,
			expectedUsed:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, used := parseSwapUsageDarwin(tt.input)
			if total != tt.expectedTotal || used != tt.expectedUsed {
				t.Errorf("parseSwapUsageDarwin() = (%d, %d), expected (%d, %d)",
					total, used, tt.expectedTotal, tt.expectedUsed)
			}
		})
	}
}

func TestParseSizeMb(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"6144.00M", 6144},
		{"2.5G", 2560},
		{"102400K", 100},
		{"1048576B", 1},
		{"500MB", 500},
		{"", 0},
	}
	for _, tt := range tests {
		if actual := parseSizeMb(tt.input); actual != tt.expected {
			t.Errorf("parseSizeMb(%q) = %d, expected %d", tt.input, actual, tt.expected)
		}
	}
}

func TestParseLinuxMemInfo(t *testing.T) {
	sampleMeminfo := `MemTotal:       16384000 kB
MemFree:         4096000 kB
MemAvailable:    8192000 kB
Buffers:          500000 kB
Cached:          3500000 kB
SwapTotal:       4194304 kB
SwapFree:        3145728 kB
`
	total, free, avail, swapTotal, swapUsed := parseLinuxMemInfo([]byte(sampleMeminfo))
	if total != 16000 {
		t.Errorf("total = %d, expected 16000", total)
	}
	if free != 4000 {
		t.Errorf("free = %d, expected 4000", free)
	}
	if avail != 8000 {
		t.Errorf("avail = %d, expected 8000", avail)
	}
	if swapTotal != 4096 {
		t.Errorf("swapTotal = %d, expected 4096", swapTotal)
	}
	if swapUsed != 1024 {
		t.Errorf("swapUsed = %d, expected 1024", swapUsed)
	}
}

func TestGetHostMemoryInfoLive(t *testing.T) {
	info := GetHostMemoryInfo()
	if info.PageSizeBytes != os.Getpagesize() {
		t.Errorf("PageSizeBytes = %d, expected %d", info.PageSizeBytes, os.Getpagesize())
	}
	if info.TotalMb < 0 {
		t.Errorf("TotalMb should be non-negative, got %d", info.TotalMb)
	}
}
