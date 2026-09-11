package report

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestSweepUsesVersionedMillisecondsAndStableHosts(t *testing.T) {
	results := []scanner.HostResult{{
		IP:     net.ParseIP("192.0.2.1"),
		Status: scanner.StatusOnline,
		RTT:    2500 * time.Microsecond,
		MAC:    "00:15:5d:00:62:1b",
	}}
	summary := Sweep{
		SchemaVersion: SchemaVersion,
		DurationMS:    (3 * time.Millisecond).Milliseconds(),
		Hosts:         Hosts(results),
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded["schema_version"] != float64(1) || decoded["duration_ms"] != float64(3) {
		t.Fatalf("unexpected schema or duration: %s", data)
	}
	hosts, ok := decoded["hosts"].([]any)
	if !ok || len(hosts) != 1 {
		t.Fatalf("unexpected hosts payload: %s", data)
	}
	host := hosts[0].(map[string]any)
	if host["status"] != "online" || host["rtt_ms"] != 2.5 {
		t.Fatalf("unexpected host telemetry: %v", host)
	}
	if _, leaked := host["Status"]; leaked {
		t.Fatalf("internal Go field name leaked into JSON: %s", data)
	}
}
