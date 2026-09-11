package report

import (
	"runtime"
	"strings"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// SchemaVersion identifies the public JSON output contract.
const SchemaVersion = 1

// Sweep is the JSON representation of one interface sweep.
type Sweep struct {
	SchemaVersion int      `json:"schema_version"`
	Target        string   `json:"target"`
	ScanMode      string   `json:"scan_mode"`
	TotalHosts    int      `json:"total_hosts"`
	OnlineHosts   int      `json:"online_hosts"`
	FastHosts     int      `json:"fast_hosts"`
	SlowHosts     int      `json:"slow_hosts"`
	SilentHosts   int      `json:"silent_hosts"`
	OfflineHosts  int      `json:"offline_hosts"`
	DurationMS    int64    `json:"duration_ms"`
	OutputFile    string   `json:"output_file,omitempty"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Rows          int      `json:"rows"`
	Cols          int      `json:"cols"`
	JoinedHosts   []string `json:"joined_hosts,omitempty"`
	DroppedHosts  []string `json:"dropped_hosts,omitempty"`
	Hosts         []Host   `json:"hosts,omitempty"`
}

// Host is the stable JSON representation of a scanned host.
type Host struct {
	IP       string             `json:"ip"`
	Hostname string             `json:"hostname,omitempty"`
	Status   string             `json:"status"`
	Roles    []scanner.HostRole `json:"roles,omitempty"`
	RTTMS    float64            `json:"rtt_ms,omitempty"`
	MAC      string             `json:"mac,omitempty"`
	Vendor   string             `json:"vendor,omitempty"`
	Error    string             `json:"error,omitempty"`
}

// Hosts converts internal scanner results into stable report values.
func Hosts(results []scanner.HostResult) []Host {
	hosts := make([]Host, 0, len(results))
	for _, result := range results {
		host := Host{
			IP:       result.IP.String(),
			Hostname: result.Hostname,
			Status:   strings.ToLower(result.Status.String()),
			Roles:    result.Roles,
			RTTMS:    float64(result.RTT.Nanoseconds()) / float64(time.Millisecond),
			MAC:      result.MAC,
			Vendor:   result.Vendor,
		}
		if result.Err != nil {
			host.Error = result.Err.Error()
		}
		hosts = append(hosts, host)
	}
	return hosts
}

// ScanMode describes the platform-specific reachability mechanisms.
func ScanMode() string {
	if runtime.GOOS == "windows" {
		return "icmp"
	}
	return "icmp_with_ping_and_tcp_fallback"
}

// MultiInterface is the JSON representation of a multi-interface sweep.
type MultiInterface struct {
	SchemaVersion   int         `json:"schema_version"`
	MultiInterface  bool        `json:"multi_interface"`
	InterfacesCount int         `json:"interfaces_count"`
	TotalHosts      int         `json:"total_hosts"`
	TotalActive     int         `json:"total_active"`
	TotalFast       int         `json:"total_fast"`
	TotalSlow       int         `json:"total_slow"`
	TotalSilent     int         `json:"total_silent"`
	TotalOffline    int         `json:"total_offline"`
	DurationMS      int64       `json:"duration_ms"`
	Interfaces      []Interface `json:"interfaces"`
}

// Interface contains one interface and its associated sweep report.
type Interface struct {
	Name       string              `json:"name"`
	Index      int                 `json:"index"`
	Primary    bool                `json:"primary"`
	Target     string              `json:"target"`
	IP         string              `json:"ip,omitempty"`
	Gateway    string              `json:"gateway,omitempty"`
	LinkHealth *scanner.LinkHealth `json:"link_health,omitempty"`
	Summary    Sweep               `json:"summary"`
	Hosts      []Host              `json:"hosts"`
}
