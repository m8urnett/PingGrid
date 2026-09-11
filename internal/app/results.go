package app

import "github.com/m8urnett/PingGrid/internal/scanner"

func countHostStatuses(results []scanner.HostResult) (online, highlight, slow, silent, offline int) {
	for _, result := range results {
		switch result.Status {
		case scanner.StatusOnline:
			online++
		case scanner.StatusHighlight:
			highlight++
		case scanner.StatusSlow:
			slow++
		case scanner.StatusSilent:
			silent++
		default:
			offline++
		}
	}
	return online, highlight, slow, silent, offline
}
