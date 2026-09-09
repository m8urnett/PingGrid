package iprange

import (
	"net/netip"
)

// CanonicalType represents the classified format of an IP range input token.
type CanonicalType string

const (
	TypeAlias         CanonicalType = "Alias"
	TypeRange         CanonicalType = "Range"
	TypePrefix        CanonicalType = "Prefix"
	TypePattern       CanonicalType = "Pattern"
	TypeHostname      CanonicalType = "Hostname"
	TypeSingleAddress CanonicalType = "SingleAddress"
	TypeUnknown       CanonicalType = "Unknown"
)

// ParsedToken represents a single token after classification and initial parsing.
type ParsedToken struct {
	RawInput string
	Type     CanonicalType
	Data     any
}

// Target represents an expanded individual IP address ready for execution.
type Target struct {
	Addr     netip.Addr
	RawInput string
	CIDR     string // Populated if the source was a Prefix
	Start    string // Normalized start of range/prefix
	End      string // Normalized end of range/prefix
}

// Data structures representing parsed specifications.
type SingleAddress struct{ Addr netip.Addr }
type PrefixAddress struct{ Prefix netip.Prefix }
type AddrRange struct{ Start, End netip.Addr }
type PatternSpec struct{ Raw string }
type AliasSpec struct{ Name string }
type HostnameSpec struct{ Name string }
