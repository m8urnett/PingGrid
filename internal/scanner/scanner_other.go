//go:build !windows

package scanner

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"

	toolkitexec "github.com/m8urnett/toolkit/exec"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
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
	if conn, err := icmp.ListenPacket("udp4", "0.0.0.0"); err == nil {
		_ = conn.Close()
		p.canUDP = true
	}

	// Test if raw ICMP socket is supported (running as root or CAP_NET_RAW on Linux)
	if conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0"); err == nil {
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
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1. Try unprivileged UDP ICMP or raw ICMP socket
	if p.canUDP || p.canRaw {
		rtt, err := p.pingICMP(attemptCtx, ip4, timeout)
		if err == nil {
			return rtt, nil
		}
	}

	// 2. Fallback to system ping command if socket denied
	if p.hasPing {
		rtt, err := p.pingCommand(attemptCtx, ip4, timeout)
		if err == nil {
			return rtt, nil
		}
	}

	// 3. Fallback to TCP port probe
	return fallbackPing(attemptCtx, ip, timeout)
}

func (p *nonWinPinger) pingICMP(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	network := "udp4"
	if !p.canUDP && p.canRaw {
		network = "ip4:icmp"
	}

	listenAddress := "0.0.0.0"
	conn, err := icmp.ListenPacket(network, listenAddress)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = conn.Close()
	}()

	seq := uint16(atomic.AddUint32(&p.seqNumber, 1) & 0xffff)
	id := p.id

	request := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(id),
			Seq:  int(seq),
			Data: []byte("PingGridSweepPlatformPingV030"),
		},
	}
	msg, err := request.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("marshal ICMP request: %w", err)
	}

	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
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

		n, _, err := conn.ReadFrom(replyBuf)
		if err != nil {
			return 0, err
		}

		elapsed := time.Since(start)

		reply, err := icmp.ParseMessage(1, replyBuf[:n])
		if err != nil || reply.Type != ipv4.ICMPTypeEchoReply {
			continue
		}
		echo, ok := reply.Body.(*icmp.Echo)
		if ok && echo.Seq == int(seq) {
			return elapsed, nil
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
