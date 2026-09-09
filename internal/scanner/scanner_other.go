//go:build !windows

package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"

	toolkitexec "github.com/m8urnett/PingGrid/internal/toolkit/exec"
)

type nonWinPinger struct {
	canUDP    bool
	canRaw    bool
	hasPing   bool
	seqNumber uint32
	id        uint16
}

// NewPlatformPinger returns a cross-platform pinger for Linux and macOS.
func NewPlatformPinger() Pinger {
	p := &nonWinPinger{
		id: uint16(rand.Intn(0xffff)),
	}

	// Test if unprivileged UDP ICMP is supported (macOS default, Linux with ping_group_range)
	if conn, err := net.ListenPacket("udp4", "0.0.0.0:0"); err == nil {
		_ = conn.Close()
		p.canUDP = true
	}

	// Test if raw ICMP socket is supported (running as root or CAP_NET_RAW on Linux)
	if conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0"); err == nil {
		_ = conn.Close()
		p.canRaw = true
	}

	// Check if system ping command is available as fallback
	if _, err := exec.LookPath("ping"); err == nil {
		p.hasPing = true
	}

	return p
}

func (p *nonWinPinger) Close() error {
	return nil
}

func (p *nonWinPinger) Ping(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("non-IPv4 address: %v", ip)
	}

	// 1. Try unprivileged UDP ICMP or raw ICMP socket
	if p.canUDP || p.canRaw {
		rtt, err := p.pingICMP(ctx, ip4, timeout)
		if err == nil {
			return rtt, nil
		}
	}

	// 2. Fallback to system ping command if socket denied
	if p.hasPing {
		rtt, err := p.pingCommand(ctx, ip4, timeout)
		if err == nil {
			return rtt, nil
		}
	}

	// 3. Fallback to TCP port probe
	return fallbackPing(ctx, ip, timeout)
}

func (p *nonWinPinger) pingICMP(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	network := "udp4"
	if !p.canUDP && p.canRaw {
		network = "ip4:icmp"
	}

	conn, err := net.ListenPacket(network, "0.0.0.0:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = conn.Close()
	}()

	seq := uint16(atomic.AddUint32(&p.seqNumber, 1) & 0xffff)
	id := p.id

	// Construct 8-byte ICMP Echo Request + 24-byte payload
	payload := []byte("PingGridSweepPlatformPingV020!!")
	msg := make([]byte, 8+len(payload))
	msg[0] = 8 // ICMP Echo Request
	msg[1] = 0 // Code 0
	msg[2] = 0 // Checksum placeholder
	msg[3] = 0
	binary.BigEndian.PutUint16(msg[4:6], id)
	binary.BigEndian.PutUint16(msg[6:8], seq)
	copy(msg[8:], payload)

	cs := icmpChecksum(msg)
	binary.BigEndian.PutUint16(msg[2:4], cs)

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	var dst net.Addr
	if network == "udp4" {
		dst = &net.UDPAddr{IP: ip}
	} else {
		dst = &net.IPAddr{IP: ip}
	}

	start := time.Now()
	if _, err := conn.WriteTo(msg, dst); err != nil {
		return 0, err
	}

	replyBuf := make([]byte, 512)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		n, from, err := conn.ReadFrom(replyBuf)
		if err != nil {
			return 0, err
		}

		elapsed := time.Since(start)

		// Parse reply: handle raw IP header offset if ip4:icmp
		data := replyBuf[:n]
		if network == "ip4:icmp" && len(data) >= 20 {
			// Skip IPv4 header (usually 20 bytes, header length in lower nibble of byte 0)
			ihl := int(data[0]&0x0f) * 4
			if len(data) >= ihl {
				data = data[ihl:]
			}
		}

		if len(data) < 8 {
			continue
		}

		// Check for ICMP Echo Reply (Type 0, Code 0)
		if data[0] == 0 && data[1] == 0 {
			replySeq := binary.BigEndian.Uint16(data[6:8])
			if replySeq == seq {
				_ = from
				return elapsed, nil
			}
		}
	}
}

func (p *nonWinPinger) pingCommand(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	timeoutSec := int(timeout.Seconds())
	if timeoutSec <= 0 {
		timeoutSec = 1
	}

	var args []string
	if runtime.GOOS == "darwin" {
		// macOS ping syntax: -c 1 -t <timeout_in_sec>
		args = []string{"-c", "1", "-t", fmt.Sprintf("%d", timeoutSec), ip.String()}
	} else {
		// Linux ping syntax: -c 1 -W <timeout_in_sec>
		args = []string{"-c", "1", "-W", fmt.Sprintf("%d", timeoutSec), ip.String()}
	}

	res, err := toolkitexec.Run(ctx, "ping", args...)
	if err != nil {
		return 0, err
	}

	return res.Duration, nil
}

func icmpChecksum(data []byte) uint16 {
	var sum uint32
	length := len(data)
	for i := 0; i < length-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if length%2 != 0 {
		sum += uint32(data[length-1]) << 8
	}
	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
