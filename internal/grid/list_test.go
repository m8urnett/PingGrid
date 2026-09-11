package grid

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestRenderListActiveSorting(t *testing.T) {
	results := []scanner.HostResult{
		{
			IP:       net.ParseIP("192.168.1.50"),
			Hostname: "nas.local",
			Status:   scanner.StatusOnline,
			RTT:      50 * time.Millisecond,
		},
		{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "gateway.local",
			Status:   scanner.StatusHighlight,
			RTT:      2 * time.Millisecond,
		},
		{
			IP:       net.ParseIP("192.168.1.20"),
			Hostname: "",
			Status:   scanner.StatusOnline,
			RTT:      10 * time.Millisecond,
		},
		{
			IP:     net.ParseIP("192.168.1.99"),
			Status: scanner.StatusOffline,
			RTT:    0,
		},
	}

	out := RenderList(results, true, false, 4, nil)

	// Verify fastest host (gateway.local, 2ms) appears before 10ms and 50ms
	gwIdx := strings.Index(out, "gateway.local")
	ip20Idx := strings.Index(out, "192.168.1.20")
	nasIdx := strings.Index(out, "nas.local")

	if gwIdx == -1 || ip20Idx == -1 || nasIdx == -1 {
		t.Fatalf("Expected all active hosts to appear in list, got:\n%s", out)
	}

	if gwIdx >= ip20Idx || ip20Idx >= nasIdx {
		t.Errorf("Expected sorted by RTT ascending (gateway < 192.168.1.20 < nas): gw=%d, ip20=%d, nas=%d",
			gwIdx, ip20Idx, nasIdx)
	}

	// Offline host should not be in default list
	if strings.Contains(out, "192.168.1.99") {
		t.Errorf("Expected offline host 192.168.1.99 to be omitted when showOffline=false")
	}

	// Verify footer
	if !strings.Contains(out, "Total: 3 active hosts (out of 4 scanned).") {
		t.Errorf("Expected footer summary, got:\n%s", out)
	}
}

func TestRenderListWithOffline(t *testing.T) {
	results := []scanner.HostResult{
		{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "gateway.local",
			Status:   scanner.StatusOnline,
			RTT:      5 * time.Millisecond,
		},
		{
			IP:     net.ParseIP("192.168.1.99"),
			Status: scanner.StatusOffline,
			RTT:    0,
		},
	}

	out := RenderList(results, true, true, 2, nil)

	if !strings.Contains(out, "192.168.1.99") {
		t.Errorf("Expected offline host to be included when showOffline=true")
	}
	if !strings.Contains(out, "offline") {
		t.Errorf("Expected offline status to be printed")
	}
}

func TestRenderListPlainVsColor(t *testing.T) {
	results := []scanner.HostResult{
		{
			IP:       net.ParseIP("127.0.0.1"),
			Hostname: "localhost",
			Status:   scanner.StatusHighlight,
			RTT:      time.Millisecond,
		},
	}

	plainOut := RenderList(results, true, false, 1, nil)
	if strings.Contains(plainOut, "\033") {
		t.Errorf("Plain output should not contain ANSI escape codes: %q", plainOut)
	}

	colorOut := RenderList(results, false, false, 1, nil)
	if !strings.Contains(colorOut, "\033") {
		t.Errorf("Color output should contain ANSI escape codes: %q", colorOut)
	}
}

func TestRenderListEmpty(t *testing.T) {
	results := []scanner.HostResult{
		{
			IP:     net.ParseIP("192.168.1.99"),
			Status: scanner.StatusOffline,
		},
	}

	out := RenderList(results, true, false, 1, nil)
	if !strings.Contains(out, "No active hosts responded") {
		t.Errorf("Expected empty response message, got: %s", out)
	}
}

func TestRenderListRoles(t *testing.T) {
	results := []scanner.HostResult{
		{
			IP:       net.ParseIP("192.168.1.1"),
			Hostname: "gateway.local",
			Status:   scanner.StatusHighlight,
			Roles:    []scanner.HostRole{scanner.RoleGateway, scanner.RoleDNS},
			RTT:      time.Millisecond,
		},
		{
			IP:       net.ParseIP("192.168.1.50"),
			Hostname: "laptop.local",
			Status:   scanner.StatusOnline,
			Roles:    []scanner.HostRole{scanner.RoleLocalHost},
			RTT:      500 * time.Microsecond,
		},
		{
			IP:       net.ParseIP("192.168.1.75"),
			Hostname: "printer.local",
			Status:   scanner.StatusOnline,
			RTT:      15 * time.Millisecond,
		},
	}

	plainOut := RenderList(results, true, false, 3, nil)
	if !strings.Contains(plainOut, "ROLE") {
		t.Errorf("Expected list output to have ROLE header, got:\n%s", plainOut)
	}
	if !strings.Contains(plainOut, "[Gateway, DNS]") {
		t.Errorf("Expected list output to have [Gateway, DNS] role badge, got:\n%s", plainOut)
	}
	if !strings.Contains(plainOut, "[Me]") {
		t.Errorf("Expected list output to have [Me] role badge, got:\n%s", plainOut)
	}
	// Verify + prefix for important hosts
	if !strings.Contains(plainOut, "+ gateway.local") {
		t.Errorf("Expected + prefix for gateway.local, got:\n%s", plainOut)
	}
	if !strings.Contains(plainOut, "+ laptop.local") {
		t.Errorf("Expected + prefix for laptop.local, got:\n%s", plainOut)
	}
	// Verify normal host without roles starts with 2 spaces
	if !strings.Contains(plainOut, "  printer.local") {
		t.Errorf("Expected 2-space prefix for normal host printer.local, got:\n%s", plainOut)
	}
}
