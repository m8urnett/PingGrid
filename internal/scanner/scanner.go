package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/m8urnett/toolkit/iprange"
)

// Config holds scanner options.
type Config struct {
	TargetCIDR    string
	Count         int
	Concurrency   int
	Timeout       time.Duration
	SlowThreshold time.Duration
	GatewayIP     net.IP
	LocalHostIP   net.IP
	DNSServers    []net.IP
	DHCPServer    net.IP
	Pings         int
	BroadcastIPs  []net.IP
	DisableARP    bool
}

// DefaultScannerConfig returns sensible defaults for scanning a local network.
func DefaultScannerConfig() Config {
	return Config{
		Count:         256,
		Concurrency:   256,
		Timeout:       150 * time.Millisecond,
		SlowThreshold: 100 * time.Millisecond,
		Pings:         3,
		DisableARP:    false,
	}
}

// IsPrivateOrLocal returns true if the IP is an RFC1918 private address, loopback, or link-local.
func IsPrivateOrLocal(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// 10.0.0.0/8
	if ip4[0] == 10 {
		return true
	}
	// 172.16.0.0/12 (172.16.0.0 - 172.31.255.255)
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}
	// 127.0.0.0/8 (Loopback)
	if ip4[0] == 127 {
		return true
	}
	// 169.254.0.0/16 (Link-local)
	if ip4[0] == 169 && ip4[1] == 254 {
		return true
	}
	return false
}

// ResolveDefaultTimeout returns 150ms if all target IPs are private/local (RFC1918/loopback),
// or 400ms if any public or external IP addresses are present.
func ResolveDefaultTimeout(ips []net.IP) time.Duration {
	if len(ips) == 0 {
		return 150 * time.Millisecond
	}
	for _, ip := range ips {
		if !IsPrivateOrLocal(ip) {
			return 400 * time.Millisecond
		}
	}
	return 150 * time.Millisecond
}

// ExtractBroadcastIPs returns any subnet directed broadcast addresses or global broadcast addresses in target.
func ExtractBroadcastIPs(target string) []net.IP {
	var broadcasts []net.IP
	if target == "" {
		detectedCIDR, _, err := DetectLocalSubnet()
		if err == nil && detectedCIDR != "" {
			target = detectedCIDR
		}
	}

	tokens, err := iprange.Tokenize(target)
	if err != nil {
		return broadcasts
	}

	for _, token := range tokens {
		// Check CIDR (e.g. 192.168.1.0/24 or 10.0.0.0/28)
		if _, ipNet, err := net.ParseCIDR(token); err == nil && ipNet != nil {
			ip4 := ipNet.IP.To4()
			if ip4 != nil && len(ipNet.Mask) == 4 {
				ones, bits := ipNet.Mask.Size()
				if bits == 32 && ones <= 30 {
					bcast := make(net.IP, 4)
					for i := 0; i < 4; i++ {
						bcast[i] = ip4[i] | ^ipNet.Mask[i]
					}
					broadcasts = append(broadcasts, bcast)
				}
			}
		}
		// Also check dotted netmask (e.g. 192.168.1.0/255.255.255.0)
		parts := strings.Split(token, "/")
		if len(parts) == 2 {
			baseIP := net.ParseIP(parts[0])
			maskIP := net.ParseIP(parts[1])
			if baseIP != nil && maskIP != nil {
				base4 := baseIP.To4()
				mask4 := maskIP.To4()
				if base4 != nil && mask4 != nil {
					mask := net.IPv4Mask(mask4[0], mask4[1], mask4[2], mask4[3])
					ones, bits := mask.Size()
					if bits == 32 && ones <= 30 {
						bcast := make(net.IP, 4)
						for i := 0; i < 4; i++ {
							bcast[i] = base4[i] | ^mask[i]
						}
						broadcasts = append(broadcasts, bcast)
					}
				}
			}
		}
	}

	return broadcasts
}

// IsIPv4Broadcast checks if an IP is a broadcast address (global 255.255.255.255, standard /24 broadcast, or subnet directed broadcast).
func IsIPv4Broadcast(ip net.IP, broadcasts []net.IP) bool {
	if ip == nil {
		return false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	// 255.255.255.255 (Global limited broadcast)
	if ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 {
		return true
	}

	// Explicitly detected / extracted subnet broadcast addresses
	for _, b := range broadcasts {
		if b != nil && ip.Equal(b) {
			return true
		}
	}

	return false
}

// GenerateIPs expands a target CIDR, IP range, pattern, or address list using toolkit/iprange.
// If target is empty, it attempts to detect the active local IPv4 subnet.
func GenerateIPs(target string, count int) ([]net.IP, net.IP, error) {
	var gw net.IP
	if target == "" {
		detectedCIDR, autoGW, err := DetectLocalSubnet()
		if err == nil && detectedCIDR != "" {
			target = detectedCIDR
			gw = autoGW
		} else {
			target = "192.168.1.0/24"
		}
	}

	targets, err := iprange.ParseAndExpand(target, iprange.DefaultTargetLimit)
	if err != nil {
		return nil, nil, err
	}

	ips := make([]net.IP, 0, len(targets))
	for _, t := range targets {
		if t.Addr.Is4() {
			ips = append(ips, net.IP(t.Addr.AsSlice()))
		}
	}

	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("no valid IPv4 addresses found in %q", target)
	}

	// If a single base IP was supplied and an explicit count > 1 was requested,
	// expand sequentially from base IP.
	if count > 1 && len(targets) == 1 && len(ips) < count {
		base := ips[0].To4()
		if base != nil {
			baseInt := binary.BigEndian.Uint32(base)
			expanded := make([]net.IP, count)
			for i := 0; i < count; i++ {
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, baseInt+uint32(i))
				expanded[i] = net.IP(ipBytes)
			}
			ips = expanded
		}
	}

	if count > 0 && len(ips) > count {
		ips = ips[:count]
	}

	return ips, gw, nil
}

// DetectLocalSubnet looks up local active network interfaces to find the primary IPv4 network.
// It detects the actual subnet mask (e.g. /22, /23, /25, /28) and real gateway IP.
func DetectLocalSubnet() (string, net.IP, error) {
	ifaces, err := GetNetworkInterfaces()
	if err != nil {
		return "", nil, err
	}

	// First pass: look for the primary interface with an active IPv4 address
	for _, iface := range ifaces {
		if iface.IsPrimary && iface.IsUp && iface.IP != nil && !iface.IsLoopback {
			return iface.SweepCIDR(), iface.Gateway, nil
		}
	}

	// Second pass: any active non-loopback IPv4 interface
	for _, iface := range ifaces {
		if iface.IsUp && !iface.IsLoopback && iface.IP != nil && !iface.IP.IsLinkLocalUnicast() {
			return iface.SweepCIDR(), iface.Gateway, nil
		}
	}

	return "", nil, fmt.Errorf("no active IPv4 network interface found")
}

// ProgressFunc is called serially as each host completes.
type ProgressFunc func(completed, total int, result HostResult)

// Sweep executes the ping sweep across all provided IP addresses using a worker pool.
func Sweep(ctx context.Context, ips []net.IP, cfg Config, onProgress ProgressFunc) []HostResult {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 256
	} else if cfg.Concurrency > MaxConcurrency {
		cfg.Concurrency = MaxConcurrency
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 150 * time.Millisecond
	}
	if cfg.SlowThreshold <= 0 {
		cfg.SlowThreshold = 100 * time.Millisecond
	}
	if cfg.Pings <= 0 {
		cfg.Pings = 1
	} else if cfg.Pings > MaxPings {
		cfg.Pings = MaxPings
	}

	total := len(ips)
	results := make([]HostResult, total)
	for i, ip := range ips {
		results[i] = HostResult{
			IP:     ip,
			Status: StatusOffline,
		}
	}

	pinger := NewPlatformPinger()
	defer func() {
		_ = pinger.Close()
	}()

	var (
		completed int
		mu        sync.Mutex
	)

	var preARP map[string]ARPEntry
	if !cfg.DisableARP {
		preARP, _ = GetARPTable()
	}

	allTasks := make([]task, total)
	for i, ip := range ips {
		allTasks[i] = task{index: i, ip: ip}
	}
	runTaskBatch(ctx, allTasks, pinger, cfg, preARP, results, total, &completed, &mu, onProgress)

	if !cfg.DisableARP {
		postARP, _ := GetARPTable()
		enrichResultsWithARP(results, preARP, postARP)
	}

	return results
}

func enrichResultsWithARP(results []HostResult, preARP, postARP map[string]ARPEntry) {
	for i := range results {
		ipStr := results[i].IP.String()
		entry, postScanEntry := postARP[ipStr]
		preScanEntry, existedBeforeScan := preARP[ipStr]
		if !postScanEntry && existedBeforeScan {
			entry = preScanEntry
		}
		if !postScanEntry && !existedBeforeScan {
			continue
		}
		if entry.MAC == "" {
			continue
		}
		results[i].MAC = entry.MAC
		results[i].Vendor = entry.Vendor
		// A newly learned post-scan entry is evidence that the target answered
		// neighbor discovery. A pre-existing entry may be stale and must not
		// promote an otherwise offline host.
		if results[i].Status == StatusOffline && postScanEntry && !existedBeforeScan {
			results[i].Status = StatusSilent
		}
	}
}

type task struct {
	index int
	ip    net.IP
}

func runTaskBatch(
	ctx context.Context,
	batch []task,
	pinger Pinger,
	cfg Config,
	preARP map[string]ARPEntry,
	results []HostResult,
	total int,
	completed *int,
	mu *sync.Mutex,
	onProgress func(completed, total int, res HostResult),
) {
	if len(batch) == 0 {
		return
	}

	tasks := make(chan task, len(batch))
	for _, t := range batch {
		tasks <- t
	}
	close(tasks)

	workerCount := cfg.Concurrency
	if workerCount > len(batch) {
		workerCount = len(batch)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if IsIPv4Broadcast(t.ip, cfg.BroadcastIPs) {
					res := HostResult{
						IP:     t.ip,
						Status: StatusOffline,
					}
					mu.Lock()
					results[t.index] = res
					*completed++
					currCompleted := *completed
					mu.Unlock()

					if onProgress != nil {
						mu.Lock()
						onProgress(currCompleted, total, res)
						mu.Unlock()
					}
					continue
				}

				numPings := cfg.Pings

				var (
					bestRTT time.Duration
					lastErr error
					status  = StatusOffline
				)

				if numPings == 1 {
					rtt, err := pinger.Ping(ctx, t.ip, cfg.Timeout)
					if err == nil {
						bestRTT = rtt
						status = StatusOnline
					} else {
						lastErr = err
					}
				} else {
					type pingReply struct {
						rtt time.Duration
						err error
					}

					replyCh := make(chan pingReply, numPings)
					var pingWg sync.WaitGroup

					halfTimeout := cfg.Timeout / 2
					if halfTimeout < 25*time.Millisecond {
						halfTimeout = 25 * time.Millisecond
					}

					for p := 0; p < numPings; p++ {
						pingWg.Add(1)
						attemptTimeout := cfg.Timeout
						if p > 0 {
							attemptTimeout = halfTimeout
						}

						go func(to time.Duration) {
							defer pingWg.Done()
							rtt, err := pinger.Ping(ctx, t.ip, to)
							replyCh <- pingReply{rtt: rtt, err: err}
						}(attemptTimeout)
					}

					pingWg.Wait()
					close(replyCh)

					for rep := range replyCh {
						if rep.err == nil {
							if status == StatusOffline || rep.rtt < bestRTT {
								bestRTT = rep.rtt
							}
							status = StatusOnline
						} else if lastErr == nil {
							lastErr = rep.err
						}
					}
				}

				var roles []HostRole
				if cfg.LocalHostIP != nil && t.ip.Equal(cfg.LocalHostIP) {
					roles = append(roles, RoleLocalHost)
				}
				if cfg.GatewayIP != nil && t.ip.Equal(cfg.GatewayIP) {
					roles = append(roles, RoleGateway)
				}
				for _, dnsIP := range cfg.DNSServers {
					if dnsIP != nil && t.ip.Equal(dnsIP) {
						roles = append(roles, RoleDNS)
						break
					}
				}
				if cfg.DHCPServer != nil && t.ip.Equal(cfg.DHCPServer) {
					roles = append(roles, RoleDHCP)
				}

				if status != StatusOffline {
					if cfg.GatewayIP != nil && t.ip.Equal(cfg.GatewayIP) {
						status = StatusHighlight
					} else if bestRTT >= cfg.SlowThreshold {
						status = StatusSlow
					}
				}

				var mac, vendor string
				if preARP != nil {
					if entry, ok := preARP[t.ip.String()]; ok && entry.MAC != "" {
						mac = entry.MAC
						vendor = entry.Vendor
					}
				}

				res := HostResult{
					IP:     t.ip,
					Status: status,
					Roles:  roles,
					RTT:    bestRTT,
					MAC:    mac,
					Vendor: vendor,
					Err:    lastErr,
				}

				mu.Lock()
				results[t.index] = res
				*completed++
				currCompleted := *completed
				mu.Unlock()

				if onProgress != nil {
					mu.Lock()
					onProgress(currCompleted, total, res)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
}
