package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
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
	FastThreshold time.Duration
	SlowThreshold time.Duration
	GatewayIP     net.IP
}

// DefaultScannerConfig returns sensible defaults for scanning a local network.
func DefaultScannerConfig() Config {
	return Config{
		Count:         256,
		Concurrency:   128,
		Timeout:       400 * time.Millisecond,
		FastThreshold: 20 * time.Millisecond,
		SlowThreshold: 100 * time.Millisecond,
	}
}

// GenerateIPs expands a target CIDR, IP range, pattern, or address list using toolkit/iprange.
// If target is empty, it attempts to detect the active local IPv4 subnet.
func GenerateIPs(target string, count int) ([]net.IP, net.IP, error) {
	if count <= 0 {
		count = 256
	}

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

	targets, err := iprange.ParseAndExpand(target, count)
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

	// If a single IP was supplied and count > 1, expand sequentially from base IP
	if len(targets) == 1 && len(ips) < count {
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

	if len(ips) > count {
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
		cfg.Timeout = 500 * time.Millisecond
	}
	if cfg.FastThreshold <= 0 {
		cfg.FastThreshold = 20 * time.Millisecond
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

				rtt, err := pinger.Ping(ctx, t.ip, cfg.Timeout)
				status := StatusOffline

				if err == nil {
					if (cfg.GatewayIP != nil && t.ip.Equal(cfg.GatewayIP)) || rtt <= cfg.FastThreshold {
						status = StatusHighlight
					} else if rtt >= cfg.SlowThreshold {
						status = StatusSlow
					} else {
						status = StatusOnline
					}
				}

				res := HostResult{
					IP:     t.ip,
					Status: status,
					RTT:    rtt,
					Err:    err,
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
