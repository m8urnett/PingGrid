package scanner

import (
	"context"
	"fmt"
	"net"
	"time"
)

// fallbackPing attempts short TCP checks when ICMP mechanisms are unavailable.
func fallbackPing(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Divide the remaining budget across common discovery ports so one filtered
	// port cannot consume the entire fallback window.
	d := net.Dialer{}
	ports := []int{53, 80, 445, 22}
	deadline, _ := attemptCtx.Deadline()
	for i, port := range ports {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, context.DeadlineExceeded
		}
		portCtx, portCancel := context.WithTimeout(attemptCtx, remaining/time.Duration(len(ports)-i))
		c, err := d.DialContext(portCtx, "tcp", fmt.Sprintf("%s:%d", ip.String(), port))
		portCancel()
		if err == nil {
			_ = c.Close()
			return time.Since(start), nil
		}
		if attemptCtx.Err() != nil {
			return 0, attemptCtx.Err()
		}
	}

	return 0, fmt.Errorf("host unreachable: %v", ip)
}
