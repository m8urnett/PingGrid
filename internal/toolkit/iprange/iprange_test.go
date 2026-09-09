package iprange

import (
	"testing"
)

func TestParseAndExpand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		expected []string
		wantErr  bool
	}{
		{
			name:     "Single IP",
			input:    "192.168.1.1",
			limit:    10,
			expected: []string{"192.168.1.1"},
			wantErr:  false,
		},
		{
			name:     "CIDR Prefix",
			input:    "192.168.1.0/30",
			limit:    10,
			expected: []string{"192.168.1.0", "192.168.1.1", "192.168.1.2", "192.168.1.3"},
			wantErr:  false,
		},
		{
			name:     "Shorthand Range",
			input:    "10.0.0.1-3",
			limit:    10,
			expected: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			wantErr:  false,
		},
		{
			name:     "Full IP Range",
			input:    "10.0.0.8-10.0.0.10",
			limit:    10,
			expected: []string{"10.0.0.8", "10.0.0.9", "10.0.0.10"},
			wantErr:  false,
		},
		{
			name:     "Pattern Expansion",
			input:    "192.168.[1-2].[1-2]",
			limit:    10,
			expected: []string{"192.168.1.1", "192.168.1.2", "192.168.2.1", "192.168.2.2"},
			wantErr:  false,
		},
		{
			name:     "Deduplication Across Tokens",
			input:    "10.0.0.1, 10.0.0.1-2, 10.0.0.2",
			limit:    10,
			expected: []string{"10.0.0.1", "10.0.0.2"},
			wantErr:  false,
		},
		{
			name:    "Limit Exceeded",
			input:   "10.0.0.0/24",
			limit:   10,
			wantErr: true,
		},
		{
			name:    "Malformed IP",
			input:   "999.999.999.999",
			limit:   10,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := ParseAndExpand(tt.input, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseAndExpand(%q) err = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(targets) != len(tt.expected) {
					t.Fatalf("expected %d targets, got %d", len(tt.expected), len(targets))
				}
				for i, target := range targets {
					if target.Addr.String() != tt.expected[i] {
						t.Errorf("target[%d] = %s, want %s", i, target.Addr.String(), tt.expected[i])
					}
				}
			}
		})
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		token string
		want  CanonicalType
	}{
		{"192.168.1.1", TypeSingleAddress},
		{"2001:db8::1", TypeSingleAddress},
		{"192.168.1.0/24", TypePrefix},
		{"10.0.0.0/255.255.255.0", TypePrefix},
		{"10.0.0.1-50", TypeRange},
		{"10.0.0.1-10.0.0.50", TypeRange},
		{"192.168.1.*", TypePattern},
		{"192.168.[1-5].*", TypePattern},
		{"RFC1918", TypeAlias},
		{"router.home", TypeHostname},
	}

	for _, c := range cases {
		got := Classify(c.token)
		if got != c.want {
			t.Errorf("Classify(%q) = %s, want %s", c.token, got, c.want)
		}
	}
}
