package iprange

import (
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strings"

	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
)

var (
	// Alias
	aliasRFC1918Regex = regexp.MustCompile(`^(?i:RFC1918)$`)

	// Range
	ipv4FullRangeRegex      = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}-(?:\d{1,3}\.){3}\d{1,3}$`)
	ipv6FullRangeRegex      = regexp.MustCompile(`^[0-9A-Fa-f:]+-[0-9A-Fa-f:]+$`)
	ipv4ShorthandRangeRegex = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}-\d{1,3}$`)

	// Prefix
	ipv4CIDRRegex = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	ipv6CIDRRegex = regexp.MustCompile(`^[0-9A-Fa-f:]+/\d{1,3}$`)
	ipv4MaskRegex = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}/(?:\d{1,3}\.){3}\d{1,3}$`)
	ipv6MaskRegex = regexp.MustCompile(`^[0-9A-Fa-f:]+/[0-9A-Fa-f:]+$`)

	// Pattern
	ipv4WildcardRegex = regexp.MustCompile(`^(?:\d{1,3}|\*)(?:\.(?:\d{1,3}|\*)){3}$`)
	ipv4BracketRegex  = regexp.MustCompile(`^(?:\d{1,3}|\*|\[\d{1,3}-\d{1,3}\])(?:\.(?:\d{1,3}|\*|\[\d{1,3}-\d{1,3}\])){3}$`)
	ipv6BracketRegex  = regexp.MustCompile(`^[0-9A-Fa-f:]*\[[0-9A-Fa-f]+-[0-9A-Fa-f]+\][0-9A-Fa-f:]*$`)

	// Hostname
	hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

	// SingleAddress
	ipv4SingleAddressRegex = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}$`)
	ipv6SingleAddressRegex = regexp.MustCompile(`^[0-9A-Fa-f:]+$`)
)

// Classify identifies the canonical format category for a raw token string.
func Classify(token string) CanonicalType {
	if aliasRFC1918Regex.MatchString(token) {
		return TypeAlias
	}
	if ipv4FullRangeRegex.MatchString(token) || ipv6FullRangeRegex.MatchString(token) || ipv4ShorthandRangeRegex.MatchString(token) {
		return TypeRange
	}
	if ipv4CIDRRegex.MatchString(token) || ipv6CIDRRegex.MatchString(token) || ipv4MaskRegex.MatchString(token) || ipv6MaskRegex.MatchString(token) {
		return TypePrefix
	}
	// Check single address before patterns so pure numeric IPs are classified as SingleAddress
	if ipv4SingleAddressRegex.MatchString(token) || ipv6SingleAddressRegex.MatchString(token) {
		return TypeSingleAddress
	}
	if ipv4BracketRegex.MatchString(token) || ipv6BracketRegex.MatchString(token) || ipv4WildcardRegex.MatchString(token) {
		return TypePattern
	}
	if hostnameRegex.MatchString(token) {
		return TypeHostname
	}
	return TypeUnknown
}

// GetCanonicalData parses and validates the token into its structured representation.
func GetCanonicalData(token string, typ CanonicalType) (any, error) {
	switch typ {
	case TypeAlias:
		return AliasSpec{Name: token}, nil

	case TypePattern:
		return PatternSpec{Raw: token}, nil

	case TypeHostname:
		return HostnameSpec{Name: token}, nil

	case TypeSingleAddress:
		addr, err := netip.ParseAddr(token)
		if err != nil {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid IP address syntax", token, "Provide a valid IPv4 or IPv6 address", err)
		}
		return SingleAddress{Addr: addr}, nil

	case TypePrefix:
		// Handle CIDR (e.g. 10.0.0.0/24)
		if prefix, err := netip.ParsePrefix(token); err == nil {
			return PrefixAddress{Prefix: prefix}, nil
		}
		// Handle dotted subnet mask (e.g. 10.0.0.0/255.255.255.0)
		parts := strings.Split(token, "/")
		if len(parts) == 2 {
			addr, errAddr := netip.ParseAddr(parts[0])
			maskIP := net.ParseIP(parts[1])
			if errAddr == nil && maskIP != nil {
				var mask net.IPMask
				if addr.Is4() {
					mask = net.IPMask(maskIP.To4())
				} else {
					mask = net.IPMask(maskIP.To16())
				}
				ones, _ := mask.Size()
				return PrefixAddress{Prefix: netip.PrefixFrom(addr, ones)}, nil
			}
		}
		return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid prefix or subnet mask format", token, "Provide a valid prefix like 192.168.1.0/24", nil)

	case TypeRange:
		parts := strings.Split(token, "-")
		if len(parts) != 2 {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid range syntax", token, "Use format START-END like 10.0.0.1-50", nil)
		}

		startAddr, err := netip.ParseAddr(parts[0])
		if err != nil {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid start IP in range", parts[0], "Check IP address syntax", err)
		}

		var endAddr netip.Addr
		if strings.Contains(parts[1], ".") || strings.Contains(parts[1], ":") {
			// Full range: 10.0.0.1-10.0.0.50
			endAddr, err = netip.ParseAddr(parts[1])
		} else {
			// Shorthand range: 10.0.0.1-50
			endAddr, err = parseShorthandEnd(startAddr, parts[1])
		}

		if err != nil {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid end IP in range", parts[1], "Check IP address syntax", err)
		}

		if startAddr.Is4() != endAddr.Is4() {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Range address family mismatch", token, "Start and end IPs must both be IPv4 or both IPv6", nil)
		}
		if startAddr.Compare(endAddr) > 0 {
			return nil, errors.New(errors.ExitInput, "INPUT_INVALID", "Start IP must be less than or equal to end IP", token, "Ensure start IP <= end IP", nil)
		}

		return AddrRange{Start: startAddr, End: endAddr}, nil

	default:
		return nil, errors.New(errors.ExitInput, "INPUT_UNSUPPORTED", "Unrecognized IP address or range format", token, "Provide a valid IP, CIDR, range, pattern, or hostname", nil)
	}
}

// parseShorthandEnd converts "10.8.0.1" and "254" into "10.8.0.254"
func parseShorthandEnd(start netip.Addr, endStr string) (netip.Addr, error) {
	if !start.Is4() {
		return netip.Addr{}, fmt.Errorf("shorthand ranges only supported for IPv4")
	}
	octets := start.As4()

	var lastOctet uint8
	_, err := fmt.Sscanf(endStr, "%d", &lastOctet)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid shorthand octet %q: %w", endStr, err)
	}

	newOctets := [4]byte{octets[0], octets[1], octets[2], lastOctet}
	return netip.AddrFrom4(newOctets), nil
}
