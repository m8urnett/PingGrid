//go:build windows

package scanner

import (
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"
)

var (
	procGetIpNetTable = modiphlpapi.NewProc("GetIpNetTable")
)

func readPlatformARPCache() (map[string]string, error) {
	var size uint32
	// First call to get required buffer size (returns ERROR_INSUFFICIENT_BUFFER = 122)
	_, _, _ = procGetIpNetTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if size == 0 {
		return make(map[string]string), nil
	}

	buf := make([]byte, size)
	ret, _, callErr := procGetIpNetTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)

	// NO_ERROR is 0
	if ret != 0 {
		return nil, fmt.Errorf("GetIpNetTable failed with code %d: %w", ret, callErr)
	}

	if len(buf) < 4 {
		return make(map[string]string), nil
	}

	numEntries := binary.LittleEndian.Uint32(buf[0:4])
	table := make(map[string]string, numEntries)

	// struct MIB_IPNETROW is 24 bytes:
	// dwIndex (4), dwPhysAddrLen (4), bPhysAddr (8), dwAddr (4), dwType (4)
	const rowSize = 24
	offset := 4

	for i := uint32(0); i < numEntries && offset+rowSize <= len(buf); i++ {
		row := buf[offset : offset+rowSize]
		physAddrLen := binary.LittleEndian.Uint32(row[4:8])
		dwAddr := binary.LittleEndian.Uint32(row[16:20])
		dwType := binary.LittleEndian.Uint32(row[20:24])

		offset += rowSize

		// Filter out invalid/deleted entries (MIB_IPNET_TYPE_INVALID = 2)
		if dwType == 2 {
			continue
		}

		if physAddrLen == 0 || physAddrLen > 8 {
			continue
		}

		// dwAddr is in network byte order in memory
		ip := net.IPv4(byte(dwAddr), byte(dwAddr>>8), byte(dwAddr>>16), byte(dwAddr>>24))
		if ip.To4() == nil || ip.IsUnspecified() {
			continue
		}

		macBytes := row[8 : 8+physAddrLen]
		var macStr string
		for j, b := range macBytes {
			if j > 0 {
				macStr += ":"
			}
			macStr += fmt.Sprintf("%02x", b)
		}

		table[ip.String()] = macStr
	}

	return table, nil
}
