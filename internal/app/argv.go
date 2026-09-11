package app

import (
	"path/filepath"
	"strings"
)

func normalizeFlagToken(token string) (string, bool) {
	if token == "-" || token == "--" {
		return token, false
	}

	var raw string
	if strings.HasPrefix(token, "--") {
		raw = token[2:]
	} else if strings.HasPrefix(token, "-") || strings.HasPrefix(token, "/") {
		raw = token[1:]
	} else {
		return token, false
	}

	if raw == "" {
		return token, false
	}

	// Split name and optional value delimited by = or :
	name := raw
	val := ""
	hasVal := false
	if sep := strings.IndexAny(raw, "=:"); sep != -1 {
		name = raw[:sep]
		val = raw[sep+1:]
		hasVal = true
	}

	// Single-letter shorthands (case-sensitive where collisions exist: R vs r, W/H vs h)
	switch name {
	case "R":
		if hasVal {
			return "-R=" + val, true
		}
		return "-R", true
	case "r":
		if hasVal {
			return "-r=" + val, true
		}
		return "-r", true
	case "W":
		if hasVal {
			return "-W=" + val, true
		}
		return "-W", true
	case "H":
		if hasVal {
			return "-H=" + val, true
		}
		return "-H", true
	case "h", "?":
		return "--help", true
	case "l", "L":
		if hasVal {
			return "-l=" + val, true
		}
		return "-l", true
	case "p", "P":
		if hasVal {
			return "-p=" + val, true
		}
		return "-p", true
	case "c", "C":
		if hasVal {
			return "-c=" + val, true
		}
		return "-c", true
	case "s", "S":
		if hasVal {
			return "-s=" + val, true
		}
		return "-s", true
	case "v", "V":
		return "-v", true
	case "q", "Q":
		return "-q", true
	case "i":
		if hasVal {
			return "-i=" + val, true
		}
		return "-i", true
	case "I":
		return "-I", true
	case "A":
		return "--all-interfaces", true
	}

	// Multi-character long flag mapping (case-insensitive)
	nameLower := strings.ToLower(name)
	switch nameLower {
	case "help":
		return "--help", true
	case "version", "ver":
		return "--version", true
	case "examples", "example":
		return "--examples", true
	case "interfaces", "ifaces":
		return "--interfaces", true
	case "all-interfaces", "all-ifaces", "all":
		return "--all-interfaces", true
	case "interface", "iface":
		if hasVal {
			return "--interface=" + val, true
		}
		return "--interface", true
	case "list":
		if hasVal {
			return "--list=" + val, true
		}
		return "--list", true
	case "ascii", "text":
		if hasVal {
			return "--ascii=" + val, true
		}
		return "--ascii", true
	case "json":
		if hasVal {
			return "--json=" + val, true
		}
		return "--json", true
	case "summary":
		if hasVal {
			return "--summary=" + val, true
		}
		return "--summary", true
	case "html":
		if hasVal {
			return "--html=" + val, true
		}
		return "--html", true
	case "iframe", "embed":
		if hasVal {
			return "--iframe=" + val, true
		}
		return "--iframe", true
	case "png":
		if hasVal {
			return "--png=" + val, true
		}
		return "--png", true
	case "pings", "ping":
		if hasVal {
			return "--pings=" + val, true
		}
		return "--pings", true
	case "refresh":
		if hasVal {
			return "--refresh=" + val, true
		}
		return "--refresh", true
	case "concurrency":
		if hasVal {
			return "--concurrency=" + val, true
		}
		return "--concurrency", true
	case "timeout":
		if hasVal {
			return "--timeout=" + val, true
		}
		return "--timeout", true
	case "slow-threshold", "slowthreshold", "slow":
		if hasVal {
			return "--slow-threshold=" + val, true
		}
		return "--slow-threshold", true
	case "rows", "row":
		if hasVal {
			return "--rows=" + val, true
		}
		return "--rows", true
	case "cols", "col", "columns", "column":
		if hasVal {
			return "--cols=" + val, true
		}
		return "--cols", true
	case "width":
		if hasVal {
			return "--width=" + val, true
		}
		return "--width", true
	case "height":
		if hasVal {
			return "--height=" + val, true
		}
		return "--height", true
	case "border-width", "borderwidth":
		if hasVal {
			return "--border-width=" + val, true
		}
		return "--border-width", true
	case "scheme":
		if hasVal {
			return "--scheme=" + val, true
		}
		return "--scheme", true
	case "color-offline", "coloroffline":
		if hasVal {
			return "--color-offline=" + val, true
		}
		return "--color-offline", true
	case "color-online", "coloronline":
		if hasVal {
			return "--color-online=" + val, true
		}
		return "--color-online", true
	case "color-highlight", "colorhighlight":
		if hasVal {
			return "--color-highlight=" + val, true
		}
		return "--color-highlight", true
	case "color-slow", "colorslow":
		if hasVal {
			return "--color-slow=" + val, true
		}
		return "--color-slow", true
	case "color-border", "colorborder":
		if hasVal {
			return "--color-border=" + val, true
		}
		return "--color-border", true
	case "color-frame", "colorframe":
		if hasVal {
			return "--color-frame=" + val, true
		}
		return "--color-frame", true
	case "plain":
		return "--plain", true
	case "hud":
		if hasVal {
			return "--hud=" + val, true
		}
		return "--hud", true
	case "no-hud", "nohud":
		return "--hud=false", true
	case "arp":
		if hasVal {
			return "--arp=" + val, true
		}
		return "--arp", true
	case "no-arp", "noarp":
		return "--arp=false", true
	case "optimize-os", "optimize_os", "optimizeos":
		return "--optimize-os", true
	case "dry-run", "dry_run", "dryrun":
		return "--dry-run", true
	case "verbose":
		return "-v", true
	case "quiet":
		return "-q", true
	case "color":
		if hasVal {
			return "--color=" + val, true
		}
		return "--color", true
	case "no-color", "nocolor":
		return "--no-color", true
	case "fail-on-warning", "failonwarning":
		return "--fail-on-warning", true
	case "encoding":
		if hasVal {
			return "--encoding=" + val, true
		}
		return "--encoding", true
	case "debug-argv", "debugargv":
		return "--debug-argv", true
	case "log":
		if hasVal {
			return "--log=" + val, true
		}
		return "--log", true
	case "no-log", "nolog":
		return "--no-log", true
	case "redact":
		return "--redact", true
	}

	// Fallback for unrecognized switches starting with '/'
	if strings.HasPrefix(token, "/") {
		if len(raw) == 1 {
			return "-" + raw, false
		}
		if hasVal {
			return "--" + name + "=" + val, false
		}
		return "--" + raw, false
	}

	return token, false
}

func normalizeCLIArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	result := make([]string, 0, len(args))
	startIndex := 0
	targetSeen := false
	firstBase := strings.ToLower(filepath.Base(args[0]))
	firstIsProgram := firstBase == "pg" || firstBase == "pg.exe" || strings.HasPrefix(firstBase, "pg-") || strings.HasSuffix(firstBase, ".exe") || strings.HasSuffix(firstBase, ".test")
	if firstIsProgram {
		result = append(result, args[0])
		startIndex = 1
	}

	outputOptionFlags := map[string]bool{
		"--list":    true,
		"-l":        true,
		"--html":    true,
		"--iframe":  true,
		"--embed":   true,
		"--png":     true,
		"--json":    true,
		"--summary": true,
		"--ascii":   true,
		"--text":    true,
	}

	for i := startIndex; i < len(args); i++ {
		arg := args[i]

		// Support naked keywords
		switch strings.ToLower(arg) {
		case "examples":
			result = append(result, "--examples")
			continue
		case "version", "ver":
			result = append(result, "--version")
			continue
		case "interfaces", "ifaces":
			result = append(result, "--interfaces")
			continue
		}

		normArg, _ := normalizeFlagToken(arg)

		if outputOptionFlags[normArg] && targetSeen {
			// Once the positional target is known, the next non-switch token can
			// unambiguously be treated as this output option's destination.
			if i+1 < len(args) {
				next := args[i+1]
				_, nextIsKnownSwitch := normalizeFlagToken(next)
				if !strings.HasPrefix(next, "-") && !nextIsKnownSwitch {
					result = append(result, normArg+"="+next)
					i++ // Skip consumed filename
					continue
				}
			}
		}

		result = append(result, normArg)
		if !strings.HasPrefix(normArg, "-") {
			targetSeen = true
		}
	}

	return result
}
