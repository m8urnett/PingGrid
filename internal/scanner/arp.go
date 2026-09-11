package scanner

import (
	"net"
	"strings"
)

// ARPEntry represents a neighbor / ARP cache entry mapping an IPv4 address to a physical MAC address.
type ARPEntry struct {
	IP       net.IP
	MAC      string
	Vendor   string
	IsStatic bool
}

// IsUnicastMAC returns true if mac is a valid unicast Ethernet MAC address
// (i.e. not broadcast ff:ff:ff:ff:ff:ff, not multicast, and not all zeroes).
func IsUnicastMAC(mac string) bool {
	norm := NormalizeMAC(mac)
	if norm == "" || norm == "00:00:00:00:00:00" || norm == "ff:ff:ff:ff:ff:ff" {
		return false
	}
	// Parse first octet to check multicast bit (LSB of first byte)
	parts := strings.Split(norm, ":")
	if len(parts) != 6 {
		return false
	}
	var firstByte byte
	for i := 0; i < 2; i++ {
		c := parts[0][i]
		var val byte
		if c >= '0' && c <= '9' {
			val = c - '0'
		} else if c >= 'a' && c <= 'f' {
			val = c - 'a' + 10
		} else {
			return false
		}
		firstByte = (firstByte << 4) | val
	}
	// If lowest bit of first byte is 1, it is a multicast / broadcast address
	if (firstByte & 0x01) != 0 {
		return false
	}
	return true
}

// GetARPTable queries the operating system neighbor / ARP cache and returns
// a map of IP string -> ARPEntry.
func GetARPTable() (map[string]ARPEntry, error) {
	return getPlatformARPTable()
}
