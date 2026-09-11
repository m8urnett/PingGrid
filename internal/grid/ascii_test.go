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
	if !strings.Contains(plainOutput, "Legend: [^] Fast/Gateway: 1  [o] Online: 1  [*] Slow: 1  [·] Offline: 29  (Total: 32)") {
		t.Errorf("unexpected plain legend: %s", plainOutput)
	}
	if !strings.Contains(plainOutput, "^ ") || !strings.Contains(plainOutput, "o ") || !strings.Contains(plainOutput, "* ") {
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

func TestRenderASCIIExtraSlotsNoColor(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 2
	cfg.Cols = 4 // 8 slots total

	// Only 3 hosts provided
	results := []scanner.HostResult{
		{IP: net.IPv4(192, 168, 1, 1), Status: scanner.StatusOnline},
		{IP: net.IPv4(192, 168, 1, 2), Status: scanner.StatusHighlight},
		{IP: net.IPv4(192, 168, 1, 3), Status: scanner.StatusOffline},
	}

	plain := RenderASCII(cfg, results, true)
	if !strings.Contains(plain, "Legend: [^] Fast/Gateway: 1  [o] Online: 1  [*] Slow: 0  [·] Offline: 1  (Total: 3)") {
		t.Errorf("expected legend to count only actual 3 hosts, got:\n%s", plain)
	}

	lines := strings.Split(plain, "\n")
	var rowLines []string
	for _, l := range lines {
		if strings.Contains(l, "|") && !strings.Contains(l, "+") {
			rowLines = append(rowLines, l)
		}
	}
	if len(rowLines) != 2 {
		t.Fatalf("expected 2 grid rows, got %d", len(rowLines))
	}
	if !strings.Contains(rowLines[0], "o ^ ·   |") {
		t.Errorf("expected first row to end with empty cell space, got: %q", rowLines[0])
	}
	if !strings.Contains(rowLines[1], "|         |") {
		t.Errorf("expected second row to have all empty spaces, got: %q", rowLines[1])
	}
}

func TestRenderASCIIAutoLayout50(t *testing.T) {
	rows, cols := AutoLayout(50)
	if rows != 5 || cols != 10 {
		t.Fatalf("expected 5x10 for 50 hosts, got %dx%d", rows, cols)
	}

	cfg := DefaultConfig()
	cfg.Rows = rows
	cfg.Cols = cols

	results := make([]scanner.HostResult, 50)
	for i := range results {
		results[i] = scanner.HostResult{
			IP:     net.IPv4(192, 168, 1, byte(i+1)),
			Status: scanner.StatusOffline,
		}
	}
	results[0].Status = scanner.StatusOnline
	results[49].Status = scanner.StatusHighlight

	plain := RenderASCII(cfg, results, true)
	lines := strings.Split(plain, "\n")
	var rowLines []string
	for _, l := range lines {
		if strings.Contains(l, "|") && !strings.Contains(l, "+") {
			rowLines = append(rowLines, l)
		}
	}
	if len(rowLines) != 5 {
		t.Fatalf("expected exactly 5 rows for 50 hosts, got %d", len(rowLines))
	}
	if !strings.Contains(rowLines[0], ".1   | o") {
		t.Errorf("expected row 0 starting with .1 and 'o', got: %s", rowLines[0])
	}
	if !strings.Contains(rowLines[4], "^ |") {
		t.Errorf("expected row 4 ending with '^', got: %s", rowLines[4])
	}
}

func TestRenderASCIILocalHostMe(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 1
	cfg.Cols = 3

	results := []scanner.HostResult{
		{IP: net.IPv4(10, 8, 0, 1), Status: scanner.StatusHighlight, Roles: []scanner.HostRole{scanner.RoleGateway}},
		{IP: net.IPv4(10, 8, 0, 2), Status: scanner.StatusOnline, Roles: []scanner.HostRole{scanner.RoleLocalHost}},
		{IP: net.IPv4(10, 8, 0, 3), Status: scanner.StatusOnline},
	}

	plain := RenderASCII(cfg, results, true)
	if !strings.Contains(plain, "^ @ o") {
		t.Errorf("expected plain row with '^ @ o', got:\n%s", plain)
	}
	if !strings.Contains(plain, "Legend: [@] Me: 1  [^] Fast/Gateway: 1") {
		t.Errorf("expected legend with '[@] Me: 1', got:\n%s", plain)
	}

	colorOut := RenderASCII(cfg, results, false)
	if !strings.Contains(colorOut, "\x1b[96;1m@ \x1b[0m") {
		t.Errorf("expected ANSI color output to contain cyan '@', got:\n%s", colorOut)
	}
}

func TestRenderASCIISilent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 1
	cfg.Cols = 4

	results := []scanner.HostResult{
		{IP: net.IPv4(10, 8, 0, 1), Status: scanner.StatusHighlight},
		{IP: net.IPv4(10, 8, 0, 2), Status: scanner.StatusOnline},
		{IP: net.IPv4(10, 8, 0, 3), Status: scanner.StatusSilent, MAC: "50:9a:4c:75:3e:2f", Vendor: "MikroTik"},
		{IP: net.IPv4(10, 8, 0, 4), Status: scanner.StatusOffline},
	}

	plain := RenderASCII(cfg, results, true)
	if !strings.Contains(plain, "? ") {
		t.Errorf("expected plain row with '?', got:\n%s", plain)
	}
	if !strings.Contains(plain, "[?] Silent/Firewalled: 1") {
		t.Errorf("expected legend with '[?] Silent/Firewalled: 1', got:\n%s", plain)
	}

	colorOut := RenderASCII(cfg, results, false)
	if !strings.Contains(colorOut, "\x1b[38;5;214;1m? \x1b[0m") {
		t.Errorf("expected ANSI color output to contain amber '?', got:\n%s", colorOut)
	}
	if !strings.Contains(colorOut, "Silent/Firewalled: 1") {
		t.Errorf("expected ANSI legend to contain 'Silent/Firewalled: 1', got:\n%s", colorOut)
	}
}
