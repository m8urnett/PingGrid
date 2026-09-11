package grid

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestRenderMultiHTML(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg1.InterfaceName = "Ethernet 2"
	cfg1.Rows = 2
	cfg1.Cols = 2

	res1 := []scanner.HostResult{
		{IP: net.ParseIP("10.8.0.1"), Status: scanner.StatusOnline, RTT: 2 * time.Millisecond},
		{IP: net.ParseIP("10.8.0.2"), Status: scanner.StatusOffline},
	}

	cfg2 := DefaultConfig()
	cfg2.InterfaceName = "vEthernet (WSL)"
	cfg2.Rows = 2
	cfg2.Cols = 2

	res2 := []scanner.HostResult{
		{IP: net.ParseIP("172.28.0.1"), Status: scanner.StatusHighlight, RTT: 1 * time.Millisecond},
		{IP: net.ParseIP("172.28.0.2"), Status: scanner.StatusSilent, MAC: "00:15:5d:01:02:03"},
	}

	cfgs := []GridConfig{cfg1, cfg2}
	resultsList := [][]scanner.HostResult{res1, res2}

	htmlBytes, err := RenderMultiHTML(cfgs, resultsList, 150*time.Millisecond, 0, nil)
	if err != nil {
		t.Fatalf("RenderMultiHTML failed: %v", err)
	}

	htmlStr := string(htmlBytes)
	if !strings.Contains(htmlStr, "PingGrid Multi-Adapter Monitor") {
		t.Errorf("Expected title in multi HTML, got:\n%s", htmlStr[:300])
	}
	if !strings.Contains(htmlStr, "Ethernet 2") {
		t.Errorf("Expected 'Ethernet 2' in multi HTML")
	}
	if !strings.Contains(htmlStr, "vEthernet (WSL)") {
		t.Errorf("Expected 'vEthernet (WSL)' in multi HTML")
	}
	if !strings.Contains(htmlStr, "All Interfaces (Stacked)") {
		t.Errorf("Expected 'All Interfaces (Stacked)' tab in multi HTML")
	}
}

func TestRenderMultiImage(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg1.Rows = 2
	cfg1.Cols = 2
	res1 := []scanner.HostResult{
		{IP: net.ParseIP("10.8.0.1"), Status: scanner.StatusOnline},
		{IP: net.ParseIP("10.8.0.2"), Status: scanner.StatusOffline},
	}

	cfg2 := DefaultConfig()
	cfg2.Rows = 2
	cfg2.Cols = 2
	res2 := []scanner.HostResult{
		{IP: net.ParseIP("172.28.0.1"), Status: scanner.StatusOnline},
		{IP: net.ParseIP("172.28.0.2"), Status: scanner.StatusOffline},
	}

	cfgs := []GridConfig{cfg1, cfg2}
	resultsList := [][]scanner.HostResult{res1, res2}

	img := RenderMulti(cfgs, resultsList)
	if img == nil {
		t.Fatal("RenderMulti returned nil image")
	}

	var buf bytes.Buffer
	if err := WritePNG(img, &buf); err != nil {
		t.Fatalf("WritePNG failed on multi image: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Generated PNG buffer is empty")
	}
}

func TestRenderMultiMinimalHTMLAndPerInterfaceDurations(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg1.InterfaceName = "Ethernet"
	cfg1.Rows, cfg1.Cols = 1, 1
	cfg2 := DefaultConfig()
	cfg2.InterfaceName = "Wi-Fi"
	cfg2.Rows, cfg2.Cols = 1, 1
	results := [][]scanner.HostResult{
		{{IP: net.ParseIP("192.0.2.1"), Status: scanner.StatusHighlight}},
		{{IP: net.ParseIP("198.51.100.1"), Status: scanner.StatusOnline}},
	}
	durations := []time.Duration{10 * time.Millisecond, 25 * time.Millisecond}
	data := BuildMultiHTMLDataWithDurations([]GridConfig{cfg1, cfg2}, results, durations, 0, nil)
	if data.Interfaces[0].Duration != "10ms" || data.Interfaces[1].Duration != "25ms" || data.Duration != "25ms" {
		t.Fatalf("per-interface durations not preserved: %+v", data)
	}
	htmlBytes, err := RenderMultiMinimalHTML([]GridConfig{cfg1, cfg2}, results, durations, 0, nil)
	if err != nil {
		t.Fatalf("RenderMultiMinimalHTML failed: %v", err)
	}
	htmlText := string(htmlBytes)
	if strings.Contains(htmlText, "<button") || strings.Contains(htmlText, "filter-pill") {
		t.Fatalf("minimal multi-interface HTML contains interactive controls")
	}
	if !strings.Contains(htmlText, "Ethernet") || !strings.Contains(htmlText, "Wi-Fi") {
		t.Fatalf("minimal multi-interface HTML omitted interface labels")
	}
}
