//go:build windows

package network

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type clientSnapshot struct {
	IP           string
	MAC          string
	Hostname     string
	DeviceType   string
	DeviceIcon   string
	FirstSeen    time.Time
	LastSeen     time.Time
	LastRxBytes  int64
	LastTxBytes  int64
	CumulativeRx int64
	CumulativeTx int64
	RxMbps       float64
	TxMbps       float64
	TotalMbps    float64
}

// WindowsHotspotTracker tracks real-time statistics and connected clients for Windows Mobile Hotspot.
type WindowsHotspotTracker struct {
	mu             sync.Mutex
	prevIfaceStats InterfaceStats
	clients        map[string]*clientSnapshot
	hostnameCache  map[string]string
	hostMu         sync.RWMutex
	lastSampled    time.Time
}

var (
	defaultHotspotTracker     *WindowsHotspotTracker
	defaultHotspotTrackerOnce sync.Once
)

func GetDefaultHotspotTracker() *WindowsHotspotTracker {
	defaultHotspotTrackerOnce.Do(func() {
		defaultHotspotTracker = &WindowsHotspotTracker{
			clients:       make(map[string]*clientSnapshot),
			hostnameCache: make(map[string]string),
		}
	})
	return defaultHotspotTracker
}

// SampleHotspotStatus discovers hotspot adapter state, enumerates connected devices, and computes throughput.
func (t *WindowsHotspotTracker) SampleHotspotStatus() HotspotState {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	nowRFC := now.Format(time.RFC3339)

	// 1. Detect Hotspot Adapter
	hotspotIface, hotspotIP, err := findWindowsHotspotInterface()
	if err != nil || hotspotIface == "" {
		// Hotspot is inactive
		t.prevIfaceStats = InterfaceStats{}
		// Clear or expire clients
		t.clients = make(map[string]*clientSnapshot)
		return HotspotState{
			Active:      false,
			ClientCount: 0,
			Clients:     []HotspotClient{},
			SampledAt:   nowRFC,
			Message:     "Mobile Hotspot is inactive",
		}
	}

	// 2. Read Hotspot Interface Byte Counters
	currStats, errStats := readSpecificInterfaceStats(hotspotIface)
	var totalDlMbps, totalUlMbps float64
	var totalDlBytes, totalUlBytes int64

	elapsedSec := 2.0
	if !t.lastSampled.IsZero() {
		elapsedSec = now.Sub(t.lastSampled).Seconds()
	}
	if elapsedSec <= 0.001 {
		elapsedSec = 2.0
	}

	if errStats == nil {
		// Note on direction for Hotspot adapter:
		// TxBytes (Outbound from PC to Client) = Client Download
		// RxBytes (Inbound from Client to PC) = Client Upload
		if t.prevIfaceStats.Name == currStats.Name && t.prevIfaceStats.TxBytes > 0 {
			var txDelta, rxDelta uint64
			if currStats.TxBytes >= t.prevIfaceStats.TxBytes {
				txDelta = currStats.TxBytes - t.prevIfaceStats.TxBytes
			}
			if currStats.RxBytes >= t.prevIfaceStats.RxBytes {
				rxDelta = currStats.RxBytes - t.prevIfaceStats.RxBytes
			}
			totalDlMbps = (float64(txDelta) * 8.0) / elapsedSec / 1e6
			totalUlMbps = (float64(rxDelta) * 8.0) / elapsedSec / 1e6
		}
		t.prevIfaceStats = currStats
		totalDlBytes = int64(currStats.TxBytes)
		totalUlBytes = int64(currStats.RxBytes)
	}

	totalDlMbps = math.Round(totalDlMbps*100) / 100
	totalUlMbps = math.Round(totalUlMbps*100) / 100
	totalMbps := math.Round((totalDlMbps+totalUlMbps)*100) / 100

	// 3. Enumerate Connected Clients from ARP/Neighbor table on the Hotspot Subnet
	discovered := t.scanHotspotClients(hotspotIP)
	seenIPs := make(map[string]bool)

	var clientList []HotspotClient

	for ip, disc := range discovered {
		seenIPs[ip] = true
		snap, exists := t.clients[ip]
		if !exists {
			dtype, dicon := InferDeviceType(disc.Hostname, disc.MAC)
			snap = &clientSnapshot{
				IP:         ip,
				MAC:        disc.MAC,
				Hostname:   disc.Hostname,
				DeviceType: dtype,
				DeviceIcon: dicon,
				FirstSeen:  now,
				LastSeen:   now,
			}
			t.clients[ip] = snap
		} else {
			snap.LastSeen = now
			if disc.MAC != "" && snap.MAC == "" {
				snap.MAC = disc.MAC
			}
			if disc.Hostname != "" && (snap.Hostname == "" || strings.HasPrefix(snap.Hostname, "Device (")) {
				snap.Hostname = disc.Hostname
				snap.DeviceType, snap.DeviceIcon = InferDeviceType(disc.Hostname, snap.MAC)
			}
		}

		// Calculate client rates
		var clientDlMbps, clientUlMbps float64
		// If single device connected, client throughput matches the hotspot interface throughput
		if len(discovered) == 1 {
			clientDlMbps = totalDlMbps
			clientUlMbps = totalUlMbps
		} else if totalMbps > 0.001 {
			// Evenly distribute or track proportionally if multi-client
			clientDlMbps = math.Round((totalDlMbps/float64(len(discovered)))*100) / 100
			clientUlMbps = math.Round((totalUlMbps/float64(len(discovered)))*100) / 100
		}

		snap.RxMbps = clientDlMbps
		snap.TxMbps = clientUlMbps
		snap.TotalMbps = math.Round((clientDlMbps+clientUlMbps)*100) / 100
		snap.CumulativeRx += int64(clientDlMbps * 1e6 / 8 * elapsedSec)
		snap.CumulativeTx += int64(clientUlMbps * 1e6 / 8 * elapsedSec)

		firstSeenStr := snap.FirstSeen.Format(time.RFC3339)
		lastSeenStr := snap.LastSeen.Format(time.RFC3339)
		connDuration := formatHotspotDuration(now.Sub(snap.FirstSeen))

		clientList = append(clientList, HotspotClient{
			IP:           snap.IP,
			MAC:          snap.MAC,
			Hostname:     snap.Hostname,
			DeviceType:   snap.DeviceType,
			DeviceIcon:   snap.DeviceIcon,
			RxMbps:       snap.RxMbps,
			TxMbps:       snap.TxMbps,
			TotalMbps:    snap.TotalMbps,
			TotalRxBytes: snap.CumulativeRx,
			TotalTxBytes: snap.CumulativeTx,
			FirstSeen:    firstSeenStr,
			LastSeen:     lastSeenStr,
			Duration:     connDuration,
		})
	}

	// Clean up stale clients absent for > 30 seconds
	for ip, snap := range t.clients {
		if !seenIPs[ip] && now.Sub(snap.LastSeen) > 30*time.Second {
			delete(t.clients, ip)
		}
	}

	t.lastSampled = now

	// Aggregate and sort clients
	clientList = AggregateHotspotClients(clientList, totalMbps)

	return HotspotState{
		Active:          true,
		InterfaceName:   hotspotIface,
		GatewayIP:       hotspotIP,
		Subnet:          "192.168.137.0/24",
		ClientCount:     len(clientList),
		TotalRxMbps:     totalDlMbps,
		TotalTxMbps:     totalUlMbps,
		TotalMbps:       totalMbps,
		TotalRxBytes:    totalDlBytes,
		TotalTxBytes:    totalUlBytes,
		BandwidthMethod: "interface_measured",
		Clients:         clientList,
		SampledAt:       nowRFC,
	}
}

func formatHotspotDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// SampleActiveHotspotStatus provides global access to hotspot sampling.
func SampleActiveHotspotStatus() HotspotState {
	return GetDefaultHotspotTracker().SampleHotspotStatus()
}

// findWindowsHotspotInterface finds the active Wi-Fi direct / ICS interface and its gateway IP.
func findWindowsHotspotInterface() (ifaceName, gatewayIP string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", err
	}

	for _, ifi := range ifaces {
		// Must be Up and have multicast/broadcast
		if (ifi.Flags & net.FlagUp) == 0 {
			continue
		}

		nameLower := strings.ToLower(ifi.Name)
		addrs, aErr := ifi.Addrs()
		if aErr != nil || len(addrs) == 0 {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			ipStr := ipNet.IP.String()

			// Windows Mobile Hotspot default subnet is 192.168.137.1
			if ipStr == "192.168.137.1" {
				return ifi.Name, ipStr, nil
			}

			// Or adapter name indicates Wi-Fi Direct / Local Area Connection virtual adapter
			if strings.Contains(nameLower, "local area connection*") ||
				strings.Contains(nameLower, "wi-fi direct") ||
				strings.Contains(nameLower, "hosted network") {
				if !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() {
					return ifi.Name, ipStr, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no active hotspot interface found")
}

type discoveredPeer struct {
	IP       string
	MAC      string
	Hostname string
}

func (t *WindowsHotspotTracker) resolveHostname(ip string) string {
	t.hostMu.RLock()
	cached, ok := t.hostnameCache[ip]
	t.hostMu.RUnlock()
	if ok && cached != "" {
		return cached
	}

	// Trigger async reverse lookup if not cached
	go func(targetIP string) {
		names, err := net.LookupAddr(targetIP)
		var resolved string
		if err == nil && len(names) > 0 {
			resolved = CleanHostname(names[0], targetIP)
		} else {
			resolved = "Unknown Device"
		}
		t.hostMu.Lock()
		t.hostnameCache[targetIP] = resolved
		t.hostMu.Unlock()
	}(ip)

	if ok {
		return cached
	}
	return "Unknown Device"
}

// scanHotspotClients reads the ARP cache / neighbor table for devices in the hotspot network.
func (t *WindowsHotspotTracker) scanHotspotClients(gatewayIP string) map[string]discoveredPeer {
	peers := make(map[string]discoveredPeer)

	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return peers
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var inHotspotSection bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Interface:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ifaceIP := fields[1]
				inHotspotSection = (ifaceIP == gatewayIP || ifaceIP == "192.168.137.1")
			}
			continue
		}

		if !inHotspotSection {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		ip := fields[0]
		macRaw := fields[1]

		// Skip broadcast, multicast, header labels, and gateway self-IP
		if ip == "Internet" || ip == gatewayIP || ip == "192.168.137.1" || ip == "192.168.137.255" ||
			ip == "255.255.255.255" || strings.HasPrefix(ip, "224.") || strings.HasPrefix(ip, "239.") {
			continue
		}

		// Ensure it's a valid IP in the subnet
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil || parsedIP.To4() == nil {
			continue
		}

		formattedMAC := strings.ToUpper(strings.ReplaceAll(macRaw, "-", ":"))
		if formattedMAC == "FF:FF:FF:FF:FF:FF" || strings.HasPrefix(formattedMAC, "01:00:5E") || strings.HasPrefix(formattedMAC, "33:33:") {
			continue
		}

		hostname := t.resolveHostname(ip)

		peers[ip] = discoveredPeer{
			IP:       ip,
			MAC:      formattedMAC,
			Hostname: hostname,
		}
	}

	return peers
}

// readSpecificInterfaceStats reads cumulative byte counters for a named network adapter.
func readSpecificInterfaceStats(name string) (InterfaceStats, error) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "interface", "ipv4", "show", "subinterfaces")
	if err != nil {
		return InterfaceStats{Timestamp: time.Now()}, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	targetLower := strings.ToLower(name)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "MTU") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		ifaceName := strings.Join(fields[4:], " ")
		if strings.ToLower(ifaceName) == targetLower {
			mediaSense := fields[1]
			if mediaSense == "5" {
				return InterfaceStats{Timestamp: time.Now()}, fmt.Errorf("interface %s media disconnected", name)
			}
			rxBytes, errRx := strconv.ParseUint(fields[2], 10, 64)
			txBytes, errTx := strconv.ParseUint(fields[3], 10, 64)
			if errRx == nil && errTx == nil {
				return InterfaceStats{
					Name:      ifaceName,
					RxBytes:   rxBytes,
					TxBytes:   txBytes,
					Timestamp: time.Now(),
				}, nil
			}
		}
	}

	return InterfaceStats{Timestamp: time.Now()}, fmt.Errorf("interface %s not found in subinterfaces", name)
}
