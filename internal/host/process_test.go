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
