package host

import (
	"testing"
)

func TestParseFootprintOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name: "bytes format from footprint --format bytes",
			input: `======================================================================
zsh [56783]: 64-bit    Footprint: 1884592 B (16384 bytes per page)
======================================================================
Auxiliary data:
    phys_footprint: 1884592 B
    phys_footprint_peak: 1884592 B
`,
			expected: 1, // 1884592 / (1024*1024) = 1 MB
		},
		{
			name: "large bytes format from footprint --format bytes",
			input: `Auxiliary data:
    phys_footprint: 3708422712 B
    phys_footprint_peak: 4130490936 B
`,
			expected: 3536, // 3708422712 / 1048576 = 3536 MB
		},
		{
			name: "formatted KB output",
			input: `Auxiliary data:
    phys_footprint: 1856 KB
    phys_footprint_peak: 1856 KB
`,
			expected: 1, // 1856 / 1024 = 1 MB (NOT 1856 MB!)
		},
		{
			name: "formatted MB output",
			input: `Auxiliary data:
    phys_footprint: 129 MB
    phys_footprint_peak: 244 MB
`,
			expected: 129,
		},
		{
			name: "formatted GB output with decimal",
			input: `Auxiliary data:
    phys_footprint: 1.8 GB
    phys_footprint_peak: 2.1 GB
`,
			expected: 1843, // 1.8 * 1024 = 1843.2 -> 1843 MB
		},
		{
			name: "attached unit MB",
			input: `Auxiliary data:
    phys_footprint: 1536MB
`,
			expected: 1536,
		},
		{
			name: "unformatted bare bytes without unit",
			input: `Auxiliary data:
    phys_footprint: 209715200
`,
			expected: 200, // 209715200 / 1048576 = 200 MB
		},
		{
			name: "no footprint line",
			input: `Process 12345:
some other line
`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parseFootprintOutput(tt.input)
			if actual != tt.expected {
				t.Errorf("parseFootprintOutput() = %d, expected %d", actual, tt.expected)
			}
		})
	}
}

func TestPickQemuPid(t *testing.T) {
	const q = "/sdk/emulator/qemu/darwin-aarch64/qemu-system-aarch64-headless"
	two := "  PID COMMAND\n" +
		"  100 " + q + " -avd A -memory 1536 -port 5554 -no-snapshot-load\n" +
		"  200 " + q + " -avd B -port 5556\n" +
		"  300 /usr/bin/avdslim start qemu-system\n"
	tests := []struct {
		name, ps, port string
		want           int
	}{
		{"exact port", two, "5554", 100},
		{"second emulator", two, "5556", 200},
		// A stopped emulator must not borrow a running neighbour's PID.
		{"stopped, neighbour running", two, "5558", 0},
		{"no false substring match", "1 " + q + " -port 55540\n", "5554", 0},
		{"only portless qemu (Studio launch)", "400 " + q + " -avd C\n", "5554", 400},
		{"two portless: ambiguous", "400 " + q + " -avd C\n500 " + q + " -avd D\n", "5554", 0},
		{"own process skipped", "42 " + q + " -port 5554\n", "5554", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickQemuPid(tc.ps, tc.port, 42); got != tc.want {
				t.Errorf("pickQemuPid = %d, want %d", got, tc.want)
			}
		})
	}
}
