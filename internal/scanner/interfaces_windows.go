//go:build windows

package scanner

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	procGetBestRoute     = modiphlpapi.NewProc("GetBestRoute")
	procGetNetworkParams = modiphlpapi.NewProc("GetNetworkParams")
	procGetAdaptersInfo  = modiphlpapi.NewProc("GetAdaptersInfo")
	procGetIfEntry       = modiphlpapi.NewProc("GetIfEntry")
	procGetIfEntry2      = modiphlpapi.NewProc("GetIfEntry2")
)

type mibIPForwardRow struct {
	dwForwardDest      uint32
	dwForwardMask      uint32
	dwForwardPolicy    uint32
	dwForwardNextHop   uint32
	dwForwardIfIndex   uint32
	dwForwardType      uint32
	dwForwardProto     uint32
	dwForwardAge       uint32
	dwForwardNextHopAS uint32
	dwForwardMetric1   uint32
	dwForwardMetric2   uint32
	dwForwardMetric3   uint32
	dwForwardMetric4   uint32
	dwForwardMetric5   uint32
}

func parseCString(b []byte) string {
	for i, v := range b {
		if v == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func getPlatformDefaultGateway(dest net.IP) (net.IP, int) {
	if procGetBestRoute.Find() != nil {
		return nil, 0
	}
	var target uint32
	if dest != nil {
		ip4 := dest.To4()
		if ip4 != nil {
			target = *(*uint32)(unsafe.Pointer(&ip4[0]))
		}
	}
	var row mibIPForwardRow
	ret, _, _ := procGetBestRoute.Call(
		uintptr(target),
		0,
		uintptr(unsafe.Pointer(&row)),
	)
	if ret != 0 {
		return nil, 0
	}
	rawIP := *(*[4]byte)(unsafe.Pointer(&row.dwForwardNextHop))
	gw := net.IPv4(rawIP[0], rawIP[1], rawIP[2], rawIP[3])
	if gw.IsUnspecified() || gw.Equal(net.IPv4zero) {
		return nil, int(row.dwForwardIfIndex)
	}
	return gw, int(row.dwForwardIfIndex)
}

type winIPAddrString struct {
	Next      *winIPAddrString
	IPAddress [16]byte
	IPMask    [16]byte
	Context   uint32
}

func getPlatformDNSServers() []net.IP {
	if procGetNetworkParams.Find() != nil {
		return nil
	}
	var size uint32
	_, _, _ = procGetNetworkParams.Call(0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil
	}
	buf := make([]byte, size)
	ret, _, _ := procGetNetworkParams.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 {
		return nil
	}

	type fixedInfo struct {
		HostName         [132]byte
		DomainName       [132]byte
		CurrentDNSServer uintptr
		DNSServerList    winIPAddrString
	}
	info := (*fixedInfo)(unsafe.Pointer(&buf[0]))
	var dnsIPs []net.IP
	curr := &info.DNSServerList
	for curr != nil {
		s := parseCString(curr.IPAddress[:])
		if ip := net.ParseIP(strings.TrimSpace(s)).To4(); ip != nil && !ip.IsUnspecified() {
			dnsIPs = append(dnsIPs, ip)
		}
		curr = curr.Next
	}
	return dnsIPs
}

type mibIfRow struct {
	wszName           [256]uint16
	dwIndex           uint32
	dwType            uint32
	dwMtu             uint32
	dwSpeed           uint32
	dwPhysAddrLen     uint32
	bPhysAddr         [8]byte
	dwAdminStatus     uint32
	dwOperStatus      uint32
	dwLastChange      uint32
	dwInOctets        uint32
	dwInUcastPkts     uint32
	dwInNUcastPkts    uint32
	dwInDiscards      uint32
	dwInErrors        uint32
	dwInUnknownProtos uint32
	dwOutOctets       uint32
	dwOutUcastPkts    uint32
	dwOutNUcastPkts   uint32
	dwOutDiscards     uint32
	dwOutErrors       uint32
	dwOutQLen         uint32
	dwDescrLen        uint32
	bDescr            [256]byte
}

type mibIfRow2 struct {
	InterfaceLuid         uint64
	InterfaceIndex        uint32
	InterfaceGuid         [16]byte
	Alias                 [257]uint16
	Description           [257]uint16
	PhysicalAddressLength uint32
	PhysicalAddress       [32]byte
	PermanentAddress      [32]byte
	Mtu                   uint32
	Type                  uint32
	TunnelType            uint32
	MediaType             uint32
	PhysicalMediumType    uint32
	AccessType            uint32
	DirectionType         uint32
	Flags                 uint32
	OperStatus            uint32
	AdminStatus           uint32
	MediaConnectState     uint32
	NetworkGuid           [16]byte
	ConnectionType        uint32
	TransmitLinkSpeed     uint64
	ReceiveLinkSpeed      uint64
	InOctets              uint64
	InUcastPkts           uint64
	InNUcastPkts          uint64
	InDiscards            uint64
	InErrors              uint64
	InUnknownProtos       uint64
	InUcastOctets         uint64
	InMulticastOctets     uint64
	InBroadcastOctets     uint64
	OutOctets             uint64
	OutUcastPkts          uint64
	OutNUcastPkts         uint64
	OutDiscards           uint64
	OutErrors             uint64
	OutUcastOctets        uint64
	OutMulticastOctets    uint64
	OutBroadcastOctets    uint64
	OutQLen               uint64
}

type ipAdapterInfo struct {
	Next                *ipAdapterInfo
	ComboIndex          uint32
	AdapterName         [260]byte
	Description         [132]byte
	AddressLength       uint32
	Address             [8]byte
	Index               uint32
	Type                uint32
	DHCPEnabled         uint32
	CurrentIPAddress    *winIPAddrString
	IPAddressList       winIPAddrString
	GatewayList         winIPAddrString
	DHCPServer          winIPAddrString
	HaveWins            uint32
	PrimaryWinsServer   winIPAddrString
	SecondaryWinsServer winIPAddrString
	LeaseObtained       int64
	LeaseExpires        int64
}

func getPlatformDHCPInfo(ifIndex int) (net.IP, bool) {
	if procGetAdaptersInfo.Find() != nil {
		return nil, false
	}
	var size uint32
	_, _, _ = procGetAdaptersInfo.Call(0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil, false
	}
	buf := make([]byte, size)
	ret, _, _ := procGetAdaptersInfo.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 {
		return nil, false
	}

	curr := (*ipAdapterInfo)(unsafe.Pointer(&buf[0]))
	for curr != nil {
		if int(curr.Index) == ifIndex {
			dhcpEnabled := curr.DHCPEnabled != 0
			var dhcpIP net.IP
			if dhcpEnabled {
				s := parseCString(curr.DHCPServer.IPAddress[:])
				if ip := net.ParseIP(strings.TrimSpace(s)).To4(); ip != nil && !ip.IsUnspecified() {
					dhcpIP = ip
				}
			}
			return dhcpIP, dhcpEnabled
		}
		curr = curr.Next
	}
	return nil, false
}

func queryWindowsWiFiStatus(ifName string) (ssid, radio, band string, channel, signalPct, signalDBm int) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	matchedInterface := false
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if idx := strings.Index(l, ":"); idx != -1 {
			k := strings.TrimSpace(l[:idx])
			v := strings.TrimSpace(l[idx+1:])
			if strings.EqualFold(k, "Name") {
				matchedInterface = strings.EqualFold(v, ifName)
				continue
			}
			if !matchedInterface {
				continue
			}
			switch strings.ToLower(k) {
			case "ssid":
				ssid = v
			case "radio type":
				switch strings.ToLower(v) {
				case "802.11ax":
					radio = "Wi-Fi 6 (802.11ax)"
				case "802.11be":
					radio = "Wi-Fi 7 (802.11be)"
				case "802.11ac":
					radio = "Wi-Fi 5 (802.11ac)"
				case "802.11n":
					radio = "Wi-Fi 4 (802.11n)"
				default:
					radio = v
				}
			case "channel":
				if ch, err := strconv.Atoi(v); err == nil {
					channel = ch
					band = ChannelToBand(ch)
				}
			case "signal":
				v = strings.TrimSuffix(v, "%")
				if pct, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					signalPct = pct
					signalDBm = SignalPercentToDBm(pct)
				}
			}
		}
	}
	return
}

func getPlatformLinkHealth(ifIndex int, ifName string, mtu int, dhcpEnabled bool, dhcpServer net.IP) LinkHealth {
	lh := LinkHealth{
		MTU:         mtu,
		DHCPEnabled: dhcpEnabled,
	}

	// 1. Adapter description and DHCP lease times via GetAdaptersInfo
	var ifType uint32
	if procGetAdaptersInfo.Find() == nil {
		var size uint32
		_, _, _ = procGetAdaptersInfo.Call(0, uintptr(unsafe.Pointer(&size)))
		if size > 0 {
			buf := make([]byte, size)
			ret, _, _ := procGetAdaptersInfo.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
			if ret == 0 {
				curr := (*ipAdapterInfo)(unsafe.Pointer(&buf[0]))
				for curr != nil {
					if int(curr.Index) == ifIndex {
						lh.AdapterModel = parseCString(curr.Description[:])
						ifType = curr.Type
						if curr.DHCPEnabled != 0 {
							lh.DHCPEnabled = true
							if curr.LeaseExpires > 0 && curr.LeaseExpires < 0x70000000 {
								leaseLeft := time.Until(time.Unix(curr.LeaseExpires, 0))
								if leaseLeft > 0 {
									lh.DHCPLeaseLeft = leaseLeft
									hours := int(leaseLeft.Hours())
									mins := int(leaseLeft.Minutes()) % 60
									lh.DHCPStatus = fmt.Sprintf("Active (expires in %dh %dm)", hours, mins)
								} else {
									lh.DHCPStatus = "Active (expired)"
								}
							} else {
								lh.DHCPStatus = "Active"
							}
						} else {
							lh.DHCPStatus = "Static IP"
						}
						break
					}
					curr = curr.Next
				}
			}
		}
	}

	if !lh.DHCPEnabled && lh.DHCPStatus == "" {
		lh.DHCPStatus = "Static IP"
	}

	// 2. Link speed & MTU via GetIfEntry2 or GetIfEntry
	var speedBps uint64
	if procGetIfEntry2.Find() == nil {
		var row2 mibIfRow2
		row2.InterfaceIndex = uint32(ifIndex)
		ret, _, _ := procGetIfEntry2.Call(uintptr(unsafe.Pointer(&row2)))
		if ret == 0 {
			speedBps = row2.TransmitLinkSpeed
			if row2.Mtu > 0 && lh.MTU == 0 {
				lh.MTU = int(row2.Mtu)
			}
			if ifType == 0 {
				ifType = row2.Type
			}
		}
	}
	if speedBps == 0 && procGetIfEntry.Find() == nil {
		var row mibIfRow
		row.dwIndex = uint32(ifIndex)
		ret, _, _ := procGetIfEntry.Call(uintptr(unsafe.Pointer(&row)))
		if ret == 0 {
			speedBps = uint64(row.dwSpeed)
			if row.dwMtu > 0 && lh.MTU == 0 {
				lh.MTU = int(row.dwMtu)
			}
			if ifType == 0 {
				ifType = row.dwType
			}
		}
	}

	if speedBps > 0 {
		lh.LinkSpeedBps = speedBps
		lh.LinkSpeedStr = FormatSpeed(speedBps)
	}

	// Duplex detection: Ethernet with >= 100Mbps link speed is standard Full Duplex
	nameLower := strings.ToLower(ifName)
	descLower := strings.ToLower(lh.AdapterModel)
	isEthernet := ifType == 6 || strings.Contains(nameLower, "ethernet") || strings.Contains(descLower, "ethernet")
	if isEthernet && speedBps >= 100_000_000 {
		lh.Duplex = "Full Duplex"
	}

	// 3. Wireless detection
	isWiFi := ifType == 71 || strings.Contains(nameLower, "wi-fi") || strings.Contains(nameLower, "wlan") ||
		strings.Contains(nameLower, "wireless") || strings.Contains(descLower, "wi-fi") || strings.Contains(descLower, "wireless")
	if isWiFi {
		lh.IsWireless = true
		ssid, radio, band, ch, sigPct, sigDBm := queryWindowsWiFiStatus(ifName)
		if ssid != "" {
			lh.SSID = ssid
			lh.RadioType = radio
			lh.Band = band
			lh.Channel = ch
			lh.SignalPercent = sigPct
			lh.SignalDBm = sigDBm
		}
	}

	return lh
}
