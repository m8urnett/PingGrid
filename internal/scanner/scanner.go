package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/m8urnett/PingGrid/internal/toolkit/iprange"
)

// Config holds scanner options.
type Config struct {
	TargetCIDR    string
	Count         int
	Concurrency   int
	Timeout       time.Duration
	SlowThreshold time.Duration
	GatewayIP     net.IP
	Pings         int
	BroadcastIPs  []net.IP
}

// DefaultScannerConfig returns sensible defaults for scanning a local network.
func DefaultScannerConfig() Config {
	return Config{
		Count:         256,
		Concurrency:   128,
		Timeout:       150 * time.Millisecond,
		SlowThreshold: 100 * time.Millisecond,
		Pings:         3,
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

	// Standard IPv4 class C / /24 broadcast (all host bits set in last octet)
	if ip4[3] == 255 {
		return true
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
func DetectLocalSubnet() (string, net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			// Derive standard 256-host /24 block encompassing local address
			base := ip4.Mask(net.CIDRMask(24, 32))
			gw := net.IPv4(base[0], base[1], base[2], 1)
			return fmt.Sprintf("%s/24", base.String()), gw, nil
		}
	}
	return "", nil, fmt.Errorf("no active IPv4 network interface found")
}

// ProgressFunc is called as each host is pinged.
type ProgressFunc func(completed, total int, result HostResult)

// Sweep executes the ping sweep across all provided IP addresses using a worker pool.
func Sweep(ctx context.Context, ips []net.IP, cfg Config, onProgress ProgressFunc) []HostResult {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 64
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 150 * time.Millisecond
	}
	if cfg.SlowThreshold <= 0 {
		cfg.SlowThreshold = 100 * time.Millisecond
	}

	total := len(ips)
	results := make([]HostResult, total)
	for i, ip := range ips {
		results[i] = HostResult{
			IP:     ip,
			Status: StatusOffline,
		}
	}

	type task struct {
		index int
		ip    net.IP
	}

	tasks := make(chan task, total)
	for i, ip := range ips {
		tasks <- task{index: i, ip: ip}
	}
	close(tasks)

	var (
		wg        sync.WaitGroup
		completed int
		mu        sync.Mutex
	)

	workerCount := cfg.Concurrency
	if workerCount > total {
		workerCount = total
	}

	pinger := NewPlatformPinger()
	defer func() {
		_ = pinger.Close()
	}()

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
					// Do not ping broadcast addresses
					res := HostResult{
						IP:     t.ip,
						Status: StatusOffline,
					}
					mu.Lock()
					results[t.index] = res
					completed++
					currCompleted := completed
					mu.Unlock()

					if onProgress != nil {
						onProgress(currCompleted, total, res)
					}
					continue
				}

				numPings := cfg.Pings
				if numPings <= 0 {
					numPings = 1
				}

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
					// Concurrent multi-ping: Ping 1 full timeout, Ping 2..N half timeout
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

				if status != StatusOffline {
					if cfg.GatewayIP != nil && t.ip.Equal(cfg.GatewayIP) {
						status = StatusHighlight
					} else if bestRTT >= cfg.SlowThreshold {
						status = StatusSlow
					}
				}

				res := HostResult{
					IP:     t.ip,
					Status: status,
					RTT:    bestRTT,
					Err:    lastErr,
				}

				mu.Lock()
				results[t.index] = res
				completed++
				currCompleted := completed
				mu.Unlock()

				if onProgress != nil {
					onProgress(currCompleted, total, res)
				}
			}
		}()
	}

	wg.Wait()
	return results
}
