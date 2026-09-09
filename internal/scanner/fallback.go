package scanner

import (
	"context"
	"fmt"
	"net"
	"time"
)

// fallbackPing attempts an ICMP socket dial, or short TCP checks if unprivileged.
func fallbackPing(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	start := time.Now()

	// Try ICMP raw socket if permitted
	conn, err := net.DialTimeout("ip4:icmp", ip.String(), timeout)
	if err == nil {
		_ = conn.Close()
		return time.Since(start), nil
	}

	// Fallback to checking common discovery ports (53, 80, 445)
	d := net.Dialer{Timeout: timeout}
	for _, port := range []int{53, 80, 445, 22} {
		c, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip.String(), port))
		if err == nil {
			_ = c.Close()
			return time.Since(start), nil
		}
	}

	return 0, fmt.Errorf("host unreachable: %v", ip)
}
