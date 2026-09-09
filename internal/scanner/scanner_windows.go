//go:build windows

package scanner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modiphlpapi         = syscall.NewLazyDLL("iphlpapi.dll")
	procIcmpCreateFile  = modiphlpapi.NewProc("IcmpCreateFile")
	procIcmpCloseHandle = modiphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = modiphlpapi.NewProc("IcmpSendEcho")
)

type winPinger struct {
	pool       sync.Pool
	mu         sync.Mutex
	allHandles []syscall.Handle
}

// NewPlatformPinger creates a Windows-native ICMP pinger using iphlpapi.dll.
// It requires NO administrative/elevated privileges on Windows and pools handles for performance.
func NewPlatformPinger() Pinger {
	p := &winPinger{}
	p.pool.New = func() any {
		h, _, _ := procIcmpCreateFile.Call()
		if h == 0 || h == ^uintptr(0) {
			invalid := syscall.InvalidHandle
			return &invalid
		}
		handle := syscall.Handle(h)
		p.mu.Lock()
		p.allHandles = append(p.allHandles, handle)
		p.mu.Unlock()
		return &handle
	}
	return p
}

func (p *winPinger) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, h := range p.allHandles {
		if h != syscall.InvalidHandle {
			_, _, _ = procIcmpCloseHandle.Call(uintptr(h))
		}
	}
	p.allHandles = nil
	return nil
}

type icmpOptionInformation struct {
	Ttl         byte
	Tos         byte
	Flags       byte
	OptionsSize byte
	_           [4]byte
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       icmpOptionInformation
}

func (p *winPinger) Ping(ctx context.Context, ip net.IP, timeout time.Duration) (time.Duration, error) {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("non-IPv4 address: %v", ip)
	}

	hObj := p.pool.Get()
	hPtr, ok := hObj.(*syscall.Handle)
	if !ok || hPtr == nil || *hPtr == syscall.InvalidHandle {
		return fallbackPing(ctx, ip, timeout)
	}
	defer p.pool.Put(hPtr)
	h := *hPtr

	// in_addr expects bytes in memory order: b0, b1, b2, b3
	destAddr := binary.LittleEndian.Uint32(ip4)

	// Send 32-byte payload
	reqData := []byte("PingGridNetworkSweepTool_v020!!")
	replyBuf := make([]byte, 256)

	timeoutMs := uint32(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = 1
	}

	startTime := time.Now()
	ret, _, callErr := procIcmpSendEcho.Call(
		uintptr(h),
		uintptr(destAddr),
		uintptr(unsafe.Pointer(&reqData[0])),
		uintptr(len(reqData)),
		0,
		uintptr(unsafe.Pointer(&replyBuf[0])),
		uintptr(len(replyBuf)),
		uintptr(timeoutMs),
	)
	elapsed := time.Since(startTime)

	if ret == 0 {
		if !errors.Is(callErr, syscall.Errno(0)) {
			return 0, callErr
		}
		return 0, errors.New("request timed out")
	}

	reply := (*icmpEchoReply)(unsafe.Pointer(&replyBuf[0]))
	if reply.Status != 0 {
		return 0, fmt.Errorf("icmp status %d", reply.Status)
	}

	rtt := elapsed
	if rtt <= 0 {
		rtt = time.Duration(reply.RoundTripTime) * time.Millisecond
	}
	return rtt, nil
}
