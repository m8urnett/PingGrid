package grid

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestRenderASCII(t *testing.T) {
	cfg := DefaultConfig()
	results := make([]scanner.HostResult, 32)
	for i := range results {
		results[i] = scanner.HostResult{
			IP:     net.IPv4(192, 168, 1, byte(i)),
			Status: scanner.StatusOffline,
		}
	}
	results[1].Status = scanner.StatusHighlight
	results[1].RTT = 2 * time.Millisecond
	results[2].Status = scanner.StatusOnline
	results[2].RTT = 25 * time.Millisecond
	results[3].Status = scanner.StatusSlow
	results[3].RTT = 150 * time.Millisecond

	cfg.Rows = 1
	cfg.Cols = 32

	// Test Plain ASCII mode
	plainOutput := RenderASCII(cfg, results, true)
	if !strings.Contains(plainOutput, "Legend: [*] Fast/Gateway: 1  [o] Online: 1  [!] Slow: 1  [·] Offline: 29  (Total: 32)") {
		t.Errorf("unexpected plain legend: %s", plainOutput)
	}
	if !strings.Contains(plainOutput, "* ") || !strings.Contains(plainOutput, "o ") || !strings.Contains(plainOutput, "! ") {
		t.Errorf("expected plain glyphs in output: %s", plainOutput)
	}

	// Test ANSI Color mode
	ansiOutput := RenderASCII(cfg, results, false)
	if !strings.Contains(ansiOutput, "\x1b[38;2;") {
		t.Errorf("expected ANSI RGB escape sequences in colored output")
	}
	if !strings.Contains(ansiOutput, "■") {
		t.Errorf("expected block characters in colored output")
	}
}

func TestRenderASCIIDelta(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 1
	cfg.Cols = 4

	results := []scanner.HostResult{
		{IP: net.IPv4(192, 168, 1, 1), Status: scanner.StatusOnline, RTT: 5 * time.Millisecond},
		{IP: net.IPv4(192, 168, 1, 2), Status: scanner.StatusOffline},
		{IP: net.IPv4(192, 168, 1, 3), Status: scanner.StatusSlow, RTT: 150 * time.Millisecond},
		{IP: net.IPv4(192, 168, 1, 4), Status: scanner.StatusOffline},
	}

	deltas := []scanner.HostDelta{
		{IP: net.IPv4(192, 168, 1, 1), Kind: scanner.DeltaJoined, New: scanner.StatusOnline},
		{IP: net.IPv4(192, 168, 1, 2), Kind: scanner.DeltaDropped, Old: scanner.StatusOnline},
		{IP: net.IPv4(192, 168, 1, 3), Kind: scanner.DeltaChanged, Old: scanner.StatusOnline, New: scanner.StatusSlow},
	}

	// Plain output should contain '+ ', '- ', '~ ' and delta legend
	plain := RenderASCIIDelta(cfg, results, deltas, true)
	if !strings.Contains(plain, "+ ") || !strings.Contains(plain, "- ") || !strings.Contains(plain, "~ ") {
		t.Errorf("expected plain delta indicators (+, -, ~), got:\n%s", plain)
	}
	if !strings.Contains(plain, "Deltas: [+] Joined: 1  [-] Dropped: 1  [~] Changed: 1") {
		t.Errorf("expected delta legend in plain output, got:\n%s", plain)
	}

	// ANSI output should contain delta glyphs and colored legend
	ansi := RenderASCIIDelta(cfg, results, deltas, false)
	if !strings.Contains(ansi, "▲") || !strings.Contains(ansi, "▼") || !strings.Contains(ansi, "~") {
		t.Errorf("expected ANSI delta glyphs (▲, ▼, ~), got:\n%s", ansi)
	}
	if !strings.Contains(ansi, "Deltas: \x1b[92;1m▲\x1b[0m Joined: 1") {
		t.Errorf("expected ANSI delta legend, got:\n%s", ansi)
	}
}

func TestASCIIBorderAlignment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 2
	cfg.Cols = 32

	results := make([]scanner.HostResult, 64)
	for i := range results {
		results[i] = scanner.HostResult{
			IP:     net.IPv4(192, 168, 1, byte(i)),
			Status: scanner.StatusOffline,
		}
	}

	plain := RenderASCII(cfg, results, true)
	lines := strings.Split(plain, "\n")

	// Filter down to border and row lines
	var topBorder, firstRow, bottomBorder string
	for _, l := range lines {
		if strings.Contains(l, "+--") {
			if topBorder == "" {
				topBorder = l
			} else {
				bottomBorder = l
			}
		} else if strings.Contains(l, "|") && firstRow == "" {
			firstRow = l
		}
	}

	if topBorder == "" || firstRow == "" || bottomBorder == "" {
		t.Fatalf("failed to locate top border, row, or bottom border in output:\n%s", plain)
	}

	runeIndex := func(s string, target rune, first bool) int {
		runes := []rune(s)
		if first {
			for i, r := range runes {
				if r == target {
					return i
				}
			}
		} else {
			for i := len(runes) - 1; i >= 0; i-- {
				if runes[i] == target {
					return i
				}
			}
		}
		return -1
	}

	topFirstPlus := runeIndex(topBorder, '+', true)
	topLastPlus := runeIndex(topBorder, '+', false)
	rowFirstPipe := runeIndex(firstRow, '|', true)
	rowLastPipe := runeIndex(firstRow, '|', false)
	botFirstPlus := runeIndex(bottomBorder, '+', true)
	botLastPlus := runeIndex(bottomBorder, '+', false)

	if topFirstPlus != rowFirstPipe {
		t.Errorf("top border opening '+' column (%d) != row opening '|' column (%d)", topFirstPlus, rowFirstPipe)
	}
	if topLastPlus != rowLastPipe {
		t.Errorf("top border closing '+' column (%d) != row closing '|' column (%d)", topLastPlus, rowLastPipe)
	}
	if botFirstPlus != rowFirstPipe {
		t.Errorf("bottom border opening '+' column (%d) != row opening '|' column (%d)", botFirstPlus, rowFirstPipe)
	}
	if botLastPlus != rowLastPipe {
		t.Errorf("bottom border closing '+' column (%d) != row closing '|' column (%d)", botLastPlus, rowLastPipe)
	}
}


