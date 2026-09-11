package grid

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestRenderHTML(t *testing.T) {
	cfg := DefaultConfig()
	results := make([]scanner.HostResult, 256)
	for i := range results {
		results[i] = scanner.HostResult{
			IP:     net.IPv4(192, 168, 1, byte(i)),
			Status: scanner.StatusOffline,
		}
	}
	results[1].Status = scanner.StatusHighlight
	results[1].RTT = 500 * time.Microsecond
	results[2].Status = scanner.StatusOnline
	results[2].RTT = 15 * time.Millisecond
	results[3].Status = scanner.StatusSlow
	results[3].RTT = 150 * time.Millisecond

	deltas := []scanner.HostDelta{
		{
			IP:   results[1].IP,
			Old:  scanner.StatusOffline,
			New:  scanner.StatusHighlight,
			Kind: scanner.DeltaJoined,
		},
	}

	content, err := RenderHTML(cfg, results, 1200*time.Millisecond, 5, deltas)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	htmlStr := string(content)

	if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
		t.Errorf("missing DOCTYPE in HTML")
	}
	if !strings.Contains(htmlStr, "PingGrid Subnet Activity") {
		t.Errorf("missing title heading in HTML")
	}
	if !strings.Contains(htmlStr, "192.168.1.1") {
		t.Errorf("missing host IP in HTML cells")
	}
	if !strings.Contains(htmlStr, HexString(cfg.ColorHighlight)) {
		t.Errorf("missing highlight color in HTML")
	}
	if !strings.Contains(htmlStr, HexString(cfg.ColorSlow)) {
		t.Errorf("missing slow color in HTML")
	}
	if !strings.Contains(htmlStr, `<meta http-equiv="refresh" content="5">`) {
		t.Errorf("missing meta refresh in HTML when refresh interval is set")
	}
	if !strings.Contains(htmlStr, "Refresh in 5s") {
		t.Errorf("missing refresh countdown banner")
	}
	if !strings.Contains(htmlStr, "+1 joined") {
		t.Errorf("missing delta badge in HTML")
	}
	if !strings.Contains(htmlStr, "cell-joined") {
		t.Errorf("missing cell-joined CSS class in HTML cell")
	}
	if !strings.Contains(htmlStr, "Recent State Changes (1)") {
		t.Errorf("missing Recent State Changes section in HTML")
	}
	if !strings.Contains(htmlStr, "copy-btn") {
		t.Errorf("missing copy active IPs button in standalone HTML")
	}
	if !strings.Contains(htmlStr, "refresh-btn") {
		t.Errorf("missing refresh button in standalone HTML")
	}
	if !strings.Contains(htmlStr, "export-json-btn") {
		t.Errorf("missing export JSON button in standalone HTML")
	}
	if !strings.Contains(htmlStr, "export-csv-btn") {
		t.Errorf("missing export CSV button in standalone HTML")
	}
	if !strings.Contains(htmlStr, "theme-btn") {
		t.Errorf("missing theme toggle button in standalone HTML")
	}
	if !strings.Contains(htmlStr, "filter-pill") {
		t.Errorf("missing filter pills in standalone HTML")
	}
}

func TestRenderMinimalHTML(t *testing.T) {
	cfg := DefaultConfig()
	results := []scanner.HostResult{
		{IP: net.IPv4(192, 168, 1, 1), Status: scanner.StatusOnline, RTT: 10 * time.Millisecond},
		{IP: net.IPv4(192, 168, 1, 2), Status: scanner.StatusOffline},
	}
	deltas := []scanner.HostDelta{
		{IP: results[0].IP, Kind: scanner.DeltaJoined, New: scanner.StatusOnline},
	}

	content, err := RenderMinimalHTML(cfg, results, 500*time.Millisecond, 3, deltas)
	if err != nil {
		t.Fatalf("RenderMinimalHTML failed: %v", err)
	}

	minStr := string(content)

	if !strings.Contains(minStr, "<!DOCTYPE html>") {
		t.Errorf("missing DOCTYPE in minimal HTML")
	}
	if !strings.Contains(minStr, "grid-canvas") {
		t.Errorf("missing grid-canvas in minimal HTML")
	}
	if !strings.Contains(minStr, `<meta http-equiv="refresh" content="3">`) {
		t.Errorf("missing meta refresh in minimal HTML")
	}
	if !strings.Contains(minStr, "cell-joined") {
		t.Errorf("missing cell-joined in minimal HTML")
	}

	// Crucial requirement: minimal iframe output must have NO buttons!
	if strings.Contains(minStr, "<button") {
		t.Errorf("minimal iframe HTML must not contain any <button> elements, found button tag")
	}
	if strings.Contains(minStr, "scale-controls") {
		t.Errorf("minimal iframe HTML must not contain scale controls")
	}
	if strings.Contains(minStr, "stats-bar") {
		t.Errorf("minimal iframe HTML must not contain stats-bar")
	}
}

func TestDeriveEmbedPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"grid.html", "grid-embed.html"},
		{"dashboard.htm", "dashboard-embed.htm"},
		{"/path/to/my-grid.html", "/path/to/my-grid-embed.html"},
		{"output", "output-embed.html"},
	}

	for _, tt := range tests {
		got := DeriveEmbedPath(tt.input)
		if got != tt.expected {
			t.Errorf("DeriveEmbedPath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRenderHTMLEmptyCells(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rows = 2
	cfg.Cols = 4 // 8 slots

	results := []scanner.HostResult{
		{IP: net.IPv4(192, 168, 1, 1), Status: scanner.StatusOnline},
		{IP: net.IPv4(192, 168, 1, 2), Status: scanner.StatusOffline},
	}

	content, err := RenderHTML(cfg, results, 100*time.Millisecond, 0, nil)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	htmlStr := string(content)
	if !strings.Contains(htmlStr, "cell-empty") {
		t.Errorf("expected cell-empty class in HTML for extra cells")
	}
	if !strings.Contains(htmlStr, "All: 2") {
		t.Errorf("expected HTML stats bar to show All: 2 (scanned hosts)")
	}
}

func TestRenderHTMLLinkHealth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InterfaceName = "Ethernet 2"
	cfg.LinkHealth = &scanner.LinkHealth{
		AdapterModel: "Microsoft Hyper-V Network Adapter #2",
		LinkSpeedStr: "1 Gbps",
		Duplex:       "Full Duplex",
		MTU:          1500,
		DHCPEnabled:  true,
		DHCPStatus:   "Active",
	}

	results := []scanner.HostResult{
		{IP: net.IPv4(10, 8, 0, 1), Status: scanner.StatusOnline},
	}

	content, err := RenderHTML(cfg, results, 50*time.Millisecond, 0, nil)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	htmlStr := string(content)
	if !strings.Contains(htmlStr, "hud-card") {
		t.Errorf("expected hud-card in HTML")
	}
	if !strings.Contains(htmlStr, "Microsoft Hyper-V Network Adapter #2") {
		t.Errorf("expected adapter model in HTML HUD")
	}
	if !strings.Contains(htmlStr, "1 Gbps Full Duplex") {
		t.Errorf("expected link speed in HTML HUD")
	}
	if !strings.Contains(htmlStr, "MTU 1500") {
		t.Errorf("expected MTU in HTML HUD")
	}
}
