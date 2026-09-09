package iprange

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
)

// DefaultTargetLimit is the fallback safeguard limit when limit <= 0.
const DefaultTargetLimit = 65536

// Tokenize splits a comma-separated range list into individual trimmed token strings.
func Tokenize(rangeList string) ([]string, error) {
	rawTokens := strings.Split(rangeList, ",")
	var tokens []string
	for _, t := range rawTokens {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Empty token in range list", rangeList, "Remove redundant trailing or adjacent commas", nil)
		}
		tokens = append(tokens, trimmed)
	}
	return tokens, nil
}

// ParseRangeList tokenizes and classifies each expression in the list.
func ParseRangeList(rangeList string) ([]ParsedToken, error) {
	tokens, err := Tokenize(rangeList)
	if err != nil {
		return nil, err
	}
	var parsedTokens []ParsedToken
	for _, t := range tokens {
		typ := Classify(t)
		data, err := GetCanonicalData(t, typ)
		if err != nil {
			return nil, err
		}
		parsedTokens = append(parsedTokens, ParsedToken{RawInput: t, Type: typ, Data: data})
	}
	return parsedTokens, nil
}

// ParseAndExpand is a high-level helper that parses a range list and expands it into deduplicated targets.
func ParseAndExpand(rangeList string, limit int) ([]Target, error) {
	tokens, err := ParseRangeList(rangeList)
	if err != nil {
		return nil, err
	}
	return ExpandTokens(tokens, limit)
}

// ExpandTokens takes classified tokens and expands them into a deduplicated slice of Targets.
// If limit > 0, it returns an error when the expanded target count exceeds the limit.
func ExpandTokens(tokens []ParsedToken, limit int) ([]Target, error) {
	if limit <= 0 {
		limit = DefaultTargetLimit
	}

	var allTargets []Target
	seen := make(map[netip.Addr]bool)

	addIP := func(ip netip.Addr, raw, cidr, start, end string) error {
		if !seen[ip] {
			if len(allTargets) >= limit {
				return errors.New(errors.ExitInput, "INPUT_TOO_LARGE",
					fmt.Sprintf("Expanded target count exceeded safety limit of %d", limit),
					raw, "Narrow your IP range or increase the limit", nil)
			}
			seen[ip] = true
			allTargets = append(allTargets, Target{
				Addr:     ip,
				RawInput: raw,
				CIDR:     cidr,
				Start:    start,
				End:      end,
			})
		}
		return nil
	}

	for _, token := range tokens {
		cidr, start, end := "", "", ""
		switch d := token.Data.(type) {
		case SingleAddress:
			if err := addIP(d.Addr, token.RawInput, cidr, start, end); err != nil {
				return nil, err
			}

		case PrefixAddress:
			cidr = d.Prefix.String()
			start = d.Prefix.Masked().Addr().String()
			end = lastAddr(d.Prefix).String()

			addr := d.Prefix.Masked().Addr()
			for d.Prefix.Contains(addr) {
				if err := addIP(addr, token.RawInput, cidr, start, end); err != nil {
					return nil, err
				}
				addr = addr.Next()
			}

		case AddrRange:
			start = d.Start.String()
			end = d.End.String()
			curr := d.Start
			for curr.Compare(d.End) <= 0 {
				if err := addIP(curr, token.RawInput, cidr, start, end); err != nil {
					return nil, err
				}
				if curr == d.End {
					break
				}
				curr = curr.Next()
			}

		case AliasSpec:
			if strings.EqualFold(d.Name, "RFC1918") {
				subnets := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
				for _, s := range subnets {
					pref := netip.MustParsePrefix(s)
					addr := pref.Addr()
					for pref.Contains(addr) {
						if err := addIP(addr, token.RawInput, cidr, start, end); err != nil {
							return nil, err
						}
						addr = addr.Next()
					}
				}
			}

		case HostnameSpec:
			resolved, err := net.LookupIP(d.Name)
			if err != nil {
				return nil, errors.New(errors.ExitInput, "INPUT_NOT_FOUND",
					fmt.Sprintf("Failed to resolve hostname %q", d.Name), d.Name, "Verify network connectivity and DNS name", err)
			}
			for _, ip := range resolved {
				if addr, ok := netip.AddrFromSlice(ip); ok {
					if err := addIP(addr.Unmap(), token.RawInput, cidr, start, end); err != nil {
						return nil, err
					}
				}
			}

		case PatternSpec:
			pIps, err := expandPattern(d.Raw)
			if err != nil {
				return nil, err
			}
			for _, ip := range pIps {
				if err := addIP(ip, token.RawInput, cidr, start, end); err != nil {
					return nil, err
				}
			}
		}
	}
	return allTargets, nil
}

func expandPattern(pattern string) ([]netip.Addr, error) {
	if strings.Contains(pattern, ".") {
		return expandIPv4Pattern(pattern)
	}
	if strings.Contains(pattern, "[") {
		return expandIPv6Pattern(pattern)
	}
	return nil, errors.New(errors.ExitInput, "INPUT_UNSUPPORTED", "Unsupported pattern format", pattern, "Use IPv4 dotted patterns like 10.0.[1-5].* or IPv6 brackets", nil)
}

func expandIPv4Pattern(pattern string) ([]netip.Addr, error) {
	segments := strings.Split(pattern, ".")
	if len(segments) != 4 {
		return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid IPv4 wildcard pattern", pattern, "IPv4 pattern must contain 4 octet segments", nil)
	}

	var grid [][]int
	for _, seg := range segments {
		vals, err := parseIPv4Segment(seg)
		if err != nil {
			return nil, err
		}
		grid = append(grid, vals)
	}

	var results []netip.Addr
	for _, a := range grid[0] {
		for _, b := range grid[1] {
			for _, c := range grid[2] {
				for _, d := range grid[3] {
					addr := netip.AddrFrom4([4]byte{byte(a), byte(b), byte(c), byte(d)})
					results = append(results, addr)
				}
			}
		}
	}
	return results, nil
}

func parseIPv4Segment(seg string) ([]int, error) {
	if seg == "*" {
		vals := make([]int, 256)
		for i := 0; i <= 255; i++ {
			vals[i] = i
		}
		return vals, nil
	}
	if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
		rangeParts := strings.Split(seg[1:len(seg)-1], "-")
		if len(rangeParts) != 2 {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid bracket range in segment", seg, "Use format [start-end] e.g. [1-10]", nil)
		}
		start, err1 := strconv.Atoi(rangeParts[0])
		end, err2 := strconv.Atoi(rangeParts[1])
		if err1 != nil || err2 != nil || start < 0 || start > 255 || end < 0 || end > 255 || start > end {
			return nil, errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Bracket range out of IPv4 bounds (0-255)", seg, "Ensure 0 <= start <= end <= 255", nil)
		}
		var vals []int
		for i := start; i <= end; i++ {
			vals = append(vals, i)
		}
		return vals, nil
	}
	val, err := strconv.Atoi(seg)
	if err != nil || val < 0 || val > 255 {
		return nil, errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Segment value out of IPv4 bounds", seg, "Octet values must be between 0 and 255", err)
	}
	return []int{val}, nil
}

func expandIPv6Pattern(pattern string) ([]netip.Addr, error) {
	startIdx := strings.Index(pattern, "[")
	endIdx := strings.Index(pattern, "]")
	if startIdx == -1 || endIdx == -1 {
		return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid IPv6 bracket pattern", pattern, "Include [start-end] hex range", nil)
	}

	rangeParts := strings.Split(pattern[startIdx+1:endIdx], "-")
	if len(rangeParts) != 2 {
		return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid IPv6 bracket range segment", pattern, "Use format [hexStart-hexEnd]", nil)
	}

	startVal, err1 := strconv.ParseUint(rangeParts[0], 16, 16)
	endVal, err2 := strconv.ParseUint(rangeParts[1], 16, 16)
	if err1 != nil || err2 != nil || startVal > endVal {
		return nil, errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Invalid IPv6 hex range values", pattern, "Ensure start <= end", nil)
	}

	var results []netip.Addr
	prefix := pattern[:startIdx]
	suffix := pattern[endIdx+1:]

	for i := startVal; i <= endVal; i++ {
		rawAddr := fmt.Sprintf("%s%x%s", prefix, i, suffix)
		addr, err := netip.ParseAddr(rawAddr)
		if err != nil {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Pattern produced invalid IPv6 address", rawAddr, "Check prefix and suffix syntax", err)
		}
		results = append(results, addr)
	}
	return results, nil
}

func lastAddr(p netip.Prefix) netip.Addr {
	p = p.Masked()
	addr := p.Addr()
	b := addr.AsSlice()
	n := p.Bits()
	for i := n; i < len(b)*8; i++ {
		b[i/8] |= 1 << (7 - i%8)
	}
	a, _ := netip.AddrFromSlice(b)
	return a
}
