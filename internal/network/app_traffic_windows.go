//go:build windows

package network

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procGetExtendedTcpTable        = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable        = iphlpapi.NewProc("GetExtendedUdpTable")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procGetProcessTimes            = kernel32.NewProc("GetProcessTimes")
	procGetProcessIoCounters       = kernel32.NewProc("GetProcessIoCounters")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
)

const (
	TCP_TABLE_OWNER_PID_ALL           = 5
	UDP_TABLE_OWNER_PID               = 1
	AF_INET                           = 2
	AF_INET6                          = 23
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
)

type ioCountersStruct struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type procSnapshot struct {
	PID          uint32
	Name         string
	Path         string
	CreationTime uint64
	RxBytes      uint64
	TxBytes      uint64
	Timestamp    time.Time
}

type WindowsAppTrafficTracker struct {
	mu           sync.Mutex
	prevSnapshot map[uint32]procSnapshot
	lastSampled  time.Time
}

var (
	defaultTracker     *WindowsAppTrafficTracker
	defaultTrackerOnce sync.Once
)

func GetDefaultAppTrafficTracker() *WindowsAppTrafficTracker {
	defaultTrackerOnce.Do(func() {
		defaultTracker = &WindowsAppTrafficTracker{
			prevSnapshot: make(map[uint32]procSnapshot),
		}
	})
	return defaultTracker
}

func queryProcessDetails(pid uint32) (name, path string, creationTime uint64, ioCounters ioCountersStruct, err error) {
	if pid == 0 {
		return "System Idle", "", 0, ioCountersStruct{}, nil
	}
	if pid == 4 {
		return "System", "", 0, ioCountersStruct{}, nil
	}

	h, _, _ := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return "", "", 0, ioCountersStruct{}, fmt.Errorf("open process failed")
	}
	defer procCloseHandle.Call(h)

	// Query Process Creation Time to safely handle PID reuse
	var createTime, exitTime, kernelTime, userTime syscall.Filetime
	ret, _, _ := procGetProcessTimes.Call(
		h,
		uintptr(unsafe.Pointer(&createTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret != 0 {
		creationTime = (uint64(createTime.HighDateTime) << 32) | uint64(createTime.LowDateTime)
	}

	// Query Image Path & Name
	var buf [1024]uint16
	size := uint32(len(buf))
	ret, _, _ = procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 {
		path = syscall.UTF16ToString(buf[:size])
		name = filepath.Base(path)
	}

	// Query I/O Transfer Counters
	ret, _, _ = procGetProcessIoCounters.Call(h, uintptr(unsafe.Pointer(&ioCounters)))
	if ret == 0 {
		err = fmt.Errorf("get io counters failed")
	}

	return name, path, creationTime, ioCounters, err
}

func getActiveSocketPIDs() map[uint32]int {
	pids := make(map[uint32]int)

	// 1. IPv4 TCP
	var size uint32
	procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 1, AF_INET, TCP_TABLE_OWNER_PID_ALL, 0)
	if size > 0 {
		buf := make([]byte, size)
		ret, _, _ := procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 1, AF_INET, TCP_TABLE_OWNER_PID_ALL, 0)
		if ret == 0 && len(buf) >= 4 {
			numEntries := int(binary.LittleEndian.Uint32(buf[0:4]))
			for i := 0; i < numEntries; i++ {
				offset := 4 + i*24
				if offset+24 <= len(buf) {
					pid := binary.LittleEndian.Uint32(buf[offset+20 : offset+24])
					if pid > 0 {
						pids[pid]++
					}
				}
			}
		}
	}

	// 2. IPv4 UDP
	var udpSize uint32
	procGetExtendedUdpTable.Call(0, uintptr(unsafe.Pointer(&udpSize)), 1, AF_INET, UDP_TABLE_OWNER_PID, 0)
	if udpSize > 0 {
		uBuf := make([]byte, udpSize)
		ret, _, _ := procGetExtendedUdpTable.Call(uintptr(unsafe.Pointer(&uBuf[0])), uintptr(unsafe.Pointer(&udpSize)), 1, AF_INET, UDP_TABLE_OWNER_PID, 0)
		if ret == 0 && len(uBuf) >= 4 {
			numEntries := int(binary.LittleEndian.Uint32(uBuf[0:4]))
			for i := 0; i < numEntries; i++ {
				offset := 4 + i*12
				if offset+12 <= len(uBuf) {
					pid := binary.LittleEndian.Uint32(uBuf[offset+8 : offset+12])
					if pid > 0 {
						pids[pid]++
					}
				}
			}
		}
	}

	// 3. IPv6 TCP
	var size6 uint32
	procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size6)), 1, AF_INET6, TCP_TABLE_OWNER_PID_ALL, 0)
	if size6 > 0 {
		buf6 := make([]byte, size6)
		ret, _, _ := procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf6[0])), uintptr(unsafe.Pointer(&size6)), 1, AF_INET6, TCP_TABLE_OWNER_PID_ALL, 0)
		if ret == 0 && len(buf6) >= 4 {
			numEntries := int(binary.LittleEndian.Uint32(buf6[0:4]))
			for i := 0; i < numEntries; i++ {
				offset := 4 + i*56
				if offset+56 <= len(buf6) {
					pid := binary.LittleEndian.Uint32(buf6[offset+52 : offset+56])
					if pid > 0 {
						pids[pid]++
					}
				}
			}
		}
	}

	// 4. IPv6 UDP
	var udpSize6 uint32
	procGetExtendedUdpTable.Call(0, uintptr(unsafe.Pointer(&udpSize6)), 1, AF_INET6, UDP_TABLE_OWNER_PID, 0)
	if udpSize6 > 0 {
		uBuf6 := make([]byte, udpSize6)
		ret, _, _ := procGetExtendedUdpTable.Call(uintptr(unsafe.Pointer(&uBuf6[0])), uintptr(unsafe.Pointer(&udpSize6)), 1, AF_INET6, UDP_TABLE_OWNER_PID, 0)
		if ret == 0 && len(uBuf6) >= 4 {
			numEntries := int(binary.LittleEndian.Uint32(uBuf6[0:4]))
			for i := 0; i < numEntries; i++ {
				offset := 4 + i*28
				if offset+28 <= len(uBuf6) {
					pid := binary.LittleEndian.Uint32(uBuf6[offset+24 : offset+28])
					if pid > 0 {
						pids[pid]++
					}
				}
			}
		}
	}

	return pids
}

// SampleApplicationTraffic queries active network sockets, compares process byte transfer deltas, and computes live bandwidth.
func (t *WindowsAppTrafficTracker) SampleApplicationTraffic() ApplicationTraffic {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	activePIDs := getActiveSocketPIDs()
	currSnapshot := make(map[uint32]procSnapshot)
	var procResults []ProcessTraffic

	elapsedSec := 1.0
	if !t.lastSampled.IsZero() {
		elapsedSec = now.Sub(t.lastSampled).Seconds()
	}
	if elapsedSec <= 0.001 {
		elapsedSec = 1.0
	}

	for pid, connCount := range activePIDs {
		name, path, createTime, io, err := queryProcessDetails(pid)
		if err != nil || name == "" {
			continue
		}

		currSnap := procSnapshot{
			PID:          pid,
			Name:         name,
			Path:         path,
			CreationTime: createTime,
			RxBytes:      io.ReadTransferCount,
			TxBytes:      io.WriteTransferCount,
			Timestamp:    now,
		}
		currSnapshot[pid] = currSnap

		// Check previous snapshot to calculate delta
		if prev, exists := t.prevSnapshot[pid]; exists {
			// Validate PID wasn't recycled for a new process
			if prev.CreationTime == createTime || prev.CreationTime == 0 || createTime == 0 {
				var rxDelta, txDelta uint64
				if currSnap.RxBytes >= prev.RxBytes {
					rxDelta = currSnap.RxBytes - prev.RxBytes
				}
				if currSnap.TxBytes >= prev.TxBytes {
					txDelta = currSnap.TxBytes - prev.TxBytes
				}

				rxMbps, txMbps, totalMbps := CalculateProcessThroughput(rxDelta, txDelta, elapsedSec)
				if totalMbps > 0.001 || rxDelta > 0 || txDelta > 0 {
					procResults = append(procResults, ProcessTraffic{
						PID:         pid,
						Name:        FriendlyProcessName(name),
						Process:     name,
						Path:        path,
						RxMbps:      round2(rxMbps),
						TxMbps:      round2(txMbps),
						TotalMbps:   round2(totalMbps),
						RxBytes:     int64(rxDelta),
						TxBytes:     int64(txDelta),
						Connections: connCount,
					})
				}
			}
		}
	}

	t.prevSnapshot = currSnapshot
	t.lastSampled = now

	return AggregateAndSortAppTraffic(procResults, 8, round1(elapsedSec))
}

// SampleActiveAppTraffic samples current application traffic using the global tracker.
func SampleActiveAppTraffic() ApplicationTraffic {
	return GetDefaultAppTrafficTracker().SampleApplicationTraffic()
}
