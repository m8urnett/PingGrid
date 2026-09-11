package scanner

import (
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"00:15:5d:00:62:1b", "00:15:5d:00:62:1b"},
		{"00-15-5D-00-62-1B", "00:15:5d:00:62:1b"},
		{"0015.5d00.621b", "00:15:5d:00:62:1b"},
		{"00155d00621b", "00:15:5d:00:62:1b"},
		{"invalid", ""},
		{"", ""},
		{"00:11:22", ""},
	}

	for _, tt := range tests {
		got := NormalizeMAC(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeMAC(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLookupVendor(t *testing.T) {
	tests := []struct {
		mac  string
		want string
	}{
		{"00:15:5d:00:62:1b", "Microsoft"},
		{"00-15-5D-00-62-1B", "Microsoft"},
		{"b8:27:eb:11:22:33", "Raspberry Pi"},
		{"00:0c:29:ab:cd:ef", "VMware"},
		{"f0:18:98:01:02:03", "Apple"},
		{"dc:ca:6d:00:00:00", ""}, // Unknown or non-matching
		{"d4:ca:6d:86:6a:e3", "MikroTik"},
		{"00:c0:b7:cc:37:da", "American Megatrends"},
		{"", ""},
	}

	for _, tt := range tests {
		got := LookupVendor(tt.mac)
		if got != tt.want {
			t.Errorf("LookupVendor(%q) = %q, want %q", tt.mac, got, tt.want)
		}
	}
}
