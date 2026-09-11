//go:build windows

package scanner

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	procGetIpNetTable = modiphlpapi.NewProc("GetIpNetTable")
)

type mibIPNetRow struct {
	dwIndex       uint32
	dwPhysAddrLen uint32
	bPhysAddr     [8]byte
	dwAddr        uint32
	dwType        uint32
}

func getPlatformARPTable() (map[string]ARPEntry, error) {
	result := make(map[string]ARPEntry)

	if procGetIpNetTable.Find() == nil {
		var size uint32
		// First call to determine required buffer size
		r1, _, _ := procGetIpNetTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
		if r1 == 122 && size > 0 { // ERROR_INSUFFICIENT_BUFFER
			buf := make([]byte, size)
			r2, _, _ := procGetIpNetTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0)
			if r2 == 0 { // NO_ERROR
				numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
				rowSize := unsafe.Sizeof(mibIPNetRow{}) // 24 bytes
				for i := uint32(0); i < numEntries; i++ {
					offset := 4 + uintptr(i)*rowSize
					if offset+rowSize > uintptr(len(buf)) {
						break
					}
					row := (*mibIPNetRow)(unsafe.Pointer(&buf[offset]))
					// dwType: 1=Other, 2=Invalid (deleted), 3=Dynamic, 4=Static
					if row.dwType == 2 || row.dwPhysAddrLen < 6 {
						continue
					}
					ip := net.IPv4(
						byte(row.dwAddr),
						byte(row.dwAddr>>8),
						byte(row.dwAddr>>16),
						byte(row.dwAddr>>24),
					)
					mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
						row.bPhysAddr[0], row.bPhysAddr[1], row.bPhysAddr[2],
						row.bPhysAddr[3], row.bPhysAddr[4], row.bPhysAddr[5],
					)
					if !IsUnicastMAC(mac) {
						continue
					}
					ipStr := ip.String()
					result[ipStr] = ARPEntry{
						IP:       ip,
						MAC:      mac,
						Vendor:   LookupVendor(mac),
						IsStatic: row.dwType == 4,
					}
				}
				if len(result) > 0 {
					return result, nil
				}
			}
		}
	}

	// Fallback to parsing `arp -a`
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "arp", "-a")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil && len(result) == 0 {
		return result, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			ip := net.ParseIP(parts[0])
			if ip == nil || ip.To4() == nil {
				continue
			}
			mac := NormalizeMAC(parts[1])
			if !IsUnicastMAC(mac) {
				continue
			}
			isStatic := strings.EqualFold(parts[2], "static")
			ipStr := ip.String()
			result[ipStr] = ARPEntry{
				IP:       ip,
				MAC:      mac,
				Vendor:   LookupVendor(mac),
				IsStatic: isStatic,
			}
		}
	}

	return result, nil
}
