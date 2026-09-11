package scanner

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

const maxDNSWorkers = 64

// ResolveHostnames concurrently performs reverse DNS lookups for active hosts.
// It skips offline hosts and bounds each lookup to the specified timeout.
func ResolveHostnames(ctx context.Context, results []HostResult, timeout time.Duration) []HostResult {
	if len(results) == 0 {
		return results
	}

	out := make([]HostResult, len(results))
	copy(out, results)

	activeCount := 0
	for i := range out {
		if out[i].Status != StatusOffline {
			activeCount++
		}
	}
	if activeCount == 0 {
		return out
	}

	workerCount := activeCount
	if workerCount > maxDNSWorkers {
		workerCount = maxDNSWorkers
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	r := net.DefaultResolver
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				lookupCtx, cancel := context.WithTimeout(ctx, timeout)
				names, err := r.LookupAddr(lookupCtx, out[idx].IP.String())
				cancel()
				if err == nil && len(names) > 0 {
					out[idx].Hostname = strings.TrimSuffix(names[0], ".")
				}
			}
		}()
	}
	for i := range out {
		if out[i].Status == StatusOffline {
			continue
		}
		select {
		case jobs <- i:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out
		}
	}
	close(jobs)
	wg.Wait()
	return out
}
