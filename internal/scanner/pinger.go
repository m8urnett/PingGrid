package scanner

import (
	"context"
	"net"
	"time"
)

// Pinger defines an interface for sending ICMP echo requests.
type Pinger interface {
	Ping(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error)
	Close() error
}
