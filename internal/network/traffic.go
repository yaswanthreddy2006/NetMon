package network

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// InterfaceStats holds cumulative byte counters for a network adapter at a point in time.
type InterfaceStats struct {
	Name      string    `json:"name"`
	RxBytes   uint64    `json:"rx_bytes"`
	TxBytes   uint64    `json:"tx_bytes"`
	Timestamp time.Time `json:"timestamp"`
}

// LiveTraffic represents calculated real-time throughput rate over an active interface.
type LiveTraffic struct {
	Interface  string  `json:"interface"`
	RxMbps     float64 `json:"rx_mbps"`
	TxMbps     float64 `json:"tx_mbps"`
	RxBytesSec float64 `json:"rx_bytes_sec"`
	TxBytesSec float64 `json:"tx_bytes_sec"`
	SampledAt  string  `json:"sampled_at"`
}

// CalculateThroughput calculates the Rx and Tx rates between two samples.
func CalculateThroughput(prev, curr InterfaceStats) (rxMbps, txMbps, rxBytesSec, txBytesSec float64) {
	dur := curr.Timestamp.Sub(prev.Timestamp).Seconds()
	if dur <= 0.0001 {
		return 0, 0, 0, 0
	}

	var rxDelta, txDelta uint64
	// Handle normal delta vs counter reset / wraparound
	if curr.RxBytes >= prev.RxBytes {
		rxDelta = curr.RxBytes - prev.RxBytes
	}
	if curr.TxBytes >= prev.TxBytes {
		txDelta = curr.TxBytes - prev.TxBytes
	}

	rxBytesSec = float64(rxDelta) / dur
	txBytesSec = float64(txDelta) / dur

	// (Bytes/sec * 8 bits) / 1,000,000 = Mbps
	rxMbps = (rxBytesSec * 8.0) / 1e6
	txMbps = (txBytesSec * 8.0) / 1e6

	rxMbps = math.Round(rxMbps*100) / 100
	txMbps = math.Round(txMbps*100) / 100
	rxBytesSec = math.Round(rxBytesSec*100) / 100
	txBytesSec = math.Round(txBytesSec*100) / 100

	return rxMbps, txMbps, rxBytesSec, txBytesSec
}

// ReadActiveInterfaceStats reads byte counters for the primary active interface.
func ReadActiveInterfaceStats() (InterfaceStats, error) {
	switch runtime.GOOS {
	case "windows":
		return readWindowsInterfaceStats()
	case "linux":
		return readLinuxInterfaceStats()
	case "darwin":
		return readDarwinInterfaceStats()
	default:
		return InterfaceStats{Timestamp: time.Now()}, fmt.Errorf("unsupported platform for traffic counters: %s", runtime.GOOS)
	}
}

// readWindowsInterfaceStats parses `netsh interface ipv4 show subinterfaces`
func readWindowsInterfaceStats() (InterfaceStats, error) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "interface", "ipv4", "show", "subinterfaces")
	if err != nil {
		return InterfaceStats{Timestamp: time.Now()}, fmt.Errorf("netsh subinterfaces failed: %w", err)
	}

	return parseWindowsSubinterfaces(string(out))
}

func parseWindowsSubinterfaces(output string) (InterfaceStats, error) {
	now := time.Now()
	var candidates []InterfaceStats

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "MTU") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		mediaSense := fields[1]
		// MediaSenseState 1 = Connected
		if mediaSense != "1" && mediaSense != "connected" {
			continue
		}

		rxBytes, errRx := strconv.ParseUint(fields[2], 10, 64)
		txBytes, errTx := strconv.ParseUint(fields[3], 10, 64)
		if errRx != nil || errTx != nil {
			continue
		}

		ifaceName := strings.Join(fields[4:], " ")
		lowerName := strings.ToLower(ifaceName)

		// Filter out loopback, teredo, isatap, virtual adapters
		if strings.Contains(lowerName, "loopback") ||
			strings.Contains(lowerName, "isatap") ||
			strings.Contains(lowerName, "teredo") {
			continue
		}

		candidates = append(candidates, InterfaceStats{
			Name:      ifaceName,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			Timestamp: now,
		})
	}

	if len(candidates) == 0 {
		return InterfaceStats{Timestamp: now}, fmt.Errorf("no active network interface found")
	}

	// Select the interface with the highest combined cumulative traffic (active primary adapter)
	selected := candidates[0]
	var maxBytes uint64 = selected.RxBytes + selected.TxBytes
	for _, c := range candidates[1:] {
		total := c.RxBytes + c.TxBytes
		if total > maxBytes {
			maxBytes = total
			selected = c
		}
	}

	return selected, nil
}

// readLinuxInterfaceStats parses /proc/net/dev
func readLinuxInterfaceStats() (InterfaceStats, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return InterfaceStats{Timestamp: time.Now()}, err
	}
	return parseLinuxProcNetDev(data)
}

func parseLinuxProcNetDev(data []byte) (InterfaceStats, error) {
	now := time.Now()
	var candidates []InterfaceStats

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" || strings.HasPrefix(iface, "docker") || strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "br-") {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rxBytes, errRx := strconv.ParseUint(fields[0], 10, 64)
		txBytes, errTx := strconv.ParseUint(fields[8], 10, 64)
		if errRx != nil || errTx != nil {
			continue
		}

		candidates = append(candidates, InterfaceStats{
			Name:      iface,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			Timestamp: now,
		})
	}

	if len(candidates) == 0 {
		return InterfaceStats{Timestamp: now}, fmt.Errorf("no active linux interface in /proc/net/dev")
	}

	selected := candidates[0]
	var maxBytes uint64 = selected.RxBytes + selected.TxBytes
	for _, c := range candidates[1:] {
		total := c.RxBytes + c.TxBytes
		if total > maxBytes {
			maxBytes = total
			selected = c
		}
	}

	return selected, nil
}

// readDarwinInterfaceStats runs netstat -ibn on macOS
func readDarwinInterfaceStats() (InterfaceStats, error) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netstat", "-ibn")
	if err != nil {
		return InterfaceStats{Timestamp: time.Now()}, err
	}
	return parseDarwinNetstat(string(out))
}

func parseDarwinNetstat(output string) (InterfaceStats, error) {
	now := time.Now()
	var candidates []InterfaceStats

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 10 || fields[0] == "Name" || strings.HasPrefix(fields[0], "lo") {
			continue
		}

		// Look for Link# rows or IP entries
		if !strings.HasPrefix(fields[2], "<Link#") {
			continue
		}

		iface := fields[0]
		rxBytes, errRx := strconv.ParseUint(fields[6], 10, 64)
		txBytes, errTx := strconv.ParseUint(fields[9], 10, 64)
		if errRx != nil || errTx != nil {
			continue
		}

		candidates = append(candidates, InterfaceStats{
			Name:      iface,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			Timestamp: now,
		})
	}

	if len(candidates) == 0 {
		return InterfaceStats{Timestamp: now}, fmt.Errorf("no active darwin interface found")
	}

	selected := candidates[0]
	var maxBytes uint64 = selected.RxBytes + selected.TxBytes
	for _, c := range candidates[1:] {
		total := c.RxBytes + c.TxBytes
		if total > maxBytes {
			maxBytes = total
			selected = c
		}
	}

	return selected, nil
}
