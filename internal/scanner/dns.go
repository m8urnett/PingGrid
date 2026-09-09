package scanner

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// ResolveHostnames concurrently performs reverse DNS lookups for active hosts.
// It skips offline hosts and bounds each lookup to the specified timeout.
func ResolveHostnames(ctx context.Context, results []HostResult, timeout time.Duration) []HostResult {
	if len(results) == 0 {
		return results
	}

	out := make([]HostResult, len(results))
	copy(out, results)

	var wg sync.WaitGroup
	r := net.DefaultResolver

	for i := range out {
		if out[i].Status == StatusOffline {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			lookupCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			names, err := r.LookupAddr(lookupCtx, out[idx].IP.String())
			if err == nil && len(names) > 0 {
				out[idx].Hostname = strings.TrimSuffix(names[0], ".")
			}
		}(i)
	}

	wg.Wait()
	return out
}
