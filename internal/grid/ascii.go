package grid

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// ansiRGB formats an RGB color as a 24-bit TrueColor foreground escape sequence.
func ansiRGB(c color.RGBA) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

const ansiReset = "\x1b[0m"

// RenderASCII renders an activity matrix to a string suitable for console display.
// When plain is true, monochrome ASCII symbols are used without ANSI escape sequences.
func RenderASCII(cfg GridConfig, results []scanner.HostResult, plain bool) string {
	return RenderASCIIDelta(cfg, results, nil, plain)
}

// RenderASCIIDelta renders an activity matrix highlighting state changes between sweeps.
func RenderASCIIDelta(cfg GridConfig, results []scanner.HostResult, deltas []scanner.HostDelta, plain bool) string {
	var sb strings.Builder

	deltaMap := make(map[string]scanner.HostDelta, len(deltas))
	var joinedCount, droppedCount, changedCount int
	for _, d := range deltas {
		deltaMap[d.IP.String()] = d
		switch d.Kind {
		case scanner.DeltaJoined:
			joinedCount++
		case scanner.DeltaDropped:
			droppedCount++
		case scanner.DeltaChanged:
			changedCount++
		}
	}

	totalSlots := cfg.Rows * cfg.Cols
	var onlineCount, fastCount, slowCount, offlineCount int

	for i := 0; i < totalSlots; i++ {
		if i < len(results) {
			switch results[i].Status {
			case scanner.StatusHighlight:
				fastCount++
			case scanner.StatusOnline:
				onlineCount++
			case scanner.StatusSlow:
				slowCount++
			default:
				offlineCount++
			}
		} else {
			offlineCount++
		}
	}

	// Build column header indicators
	sb.WriteString("\n")
	sb.WriteString("      ")
	for c := 0; c < cfg.Cols; c++ {
		if c%4 == 0 {
			sb.WriteString(fmt.Sprintf("%-2d", c))
		} else {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("      +")
	for c := 0; c < cfg.Cols; c++ {
		sb.WriteString("--")
	}
	sb.WriteString("-+\n")

	// Render rows
	for r := 0; r < cfg.Rows; r++ {
		startIdx := r * cfg.Cols
		rowLabel := fmt.Sprintf(".%-3d", startIdx)
		if startIdx < len(results) && results[startIdx].IP != nil {
			ip4 := results[startIdx].IP.To4()
			if ip4 != nil {
				rowLabel = fmt.Sprintf(".%-3d", ip4[3])
			}
		}

		sb.WriteString(fmt.Sprintf(" %4s | ", rowLabel))

		for c := 0; c < cfg.Cols; c++ {
			idx := r*cfg.Cols + c
			st := scanner.StatusOffline
			var ipStr string

			if idx < len(results) {
				st = results[idx].Status
				if results[idx].IP != nil {
					ipStr = results[idx].IP.String()
				}
			}

			d, hasDelta := deltaMap[ipStr]

			if plain {
				if hasDelta {
					switch d.Kind {
					case scanner.DeltaJoined:
						sb.WriteString("+ ")
					case scanner.DeltaDropped:
						sb.WriteString("- ")
					case scanner.DeltaChanged:
						sb.WriteString("~ ")
					default:
						sb.WriteString("· ")
					}
				} else {
					switch st {
					case scanner.StatusHighlight:
						sb.WriteString("* ")
					case scanner.StatusOnline:
						sb.WriteString("o ")
					case scanner.StatusSlow:
						sb.WriteString("! ")
					default:
						sb.WriteString("· ")
					}
				}
			} else {
				if hasDelta {
					switch d.Kind {
					case scanner.DeltaJoined:
						sb.WriteString("\x1b[92;1m▲ \x1b[0m")
					case scanner.DeltaDropped:
						sb.WriteString("\x1b[91;1m▼ \x1b[0m")
					case scanner.DeltaChanged:
						sb.WriteString("\x1b[93;1m~ \x1b[0m")
					default:
						sb.WriteString(ansiRGB(cfg.ColorOffline) + "■ " + ansiReset)
					}
				} else {
					var clr color.RGBA
					switch st {
					case scanner.StatusHighlight:
						clr = cfg.ColorHighlight
					case scanner.StatusOnline:
						clr = cfg.ColorOnline
					case scanner.StatusSlow:
						clr = cfg.ColorSlow
					default:
						clr = cfg.ColorOffline
					}
					sb.WriteString(ansiRGB(clr))
					sb.WriteString("■ ")
					sb.WriteString(ansiReset)
				}
			}
		}
		sb.WriteString("|\n")
	}

	sb.WriteString("      +")
	for c := 0; c < cfg.Cols; c++ {
		sb.WriteString("--")
	}
	sb.WriteString("-+\n\n")

	// Summary Legend
	if plain {
		sb.WriteString(fmt.Sprintf(" Legend: [*] Fast/Gateway: %d  [o] Online: %d  [!] Slow: %d  [·] Offline: %d  (Total: %d)\n",
			fastCount, onlineCount, slowCount, offlineCount, totalSlots))
		if len(deltas) > 0 {
			sb.WriteString(fmt.Sprintf(" Deltas: [+] Joined: %d  [-] Dropped: %d  [~] Changed: %d\n",
				joinedCount, droppedCount, changedCount))
		}
	} else {
		sb.WriteString(fmt.Sprintf(" Legend: %s■%s Fast/Gateway: %d  %s■%s Online: %d  %s■%s Slow: %d  %s■%s Offline: %d  (Total: %d)\n",
			ansiRGB(cfg.ColorHighlight), ansiReset, fastCount,
			ansiRGB(cfg.ColorOnline), ansiReset, onlineCount,
			ansiRGB(cfg.ColorSlow), ansiReset, slowCount,
			ansiRGB(cfg.ColorOffline), ansiReset, offlineCount,
			totalSlots))
		if len(deltas) > 0 {
			sb.WriteString(fmt.Sprintf(" Deltas: \x1b[92;1m▲\x1b[0m Joined: %d  \x1b[91;1m▼\x1b[0m Dropped: %d  \x1b[93;1m~\x1b[0m Changed: %d\n",
				joinedCount, droppedCount, changedCount))
		}
	}

	return sb.String()
}
