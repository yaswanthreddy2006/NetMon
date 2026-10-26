package network

import (
	"math"
	"sort"
	"strings"
)

// HotspotClient represents a connected device on the mobile hotspot.
type HotspotClient struct {
	IP           string  `json:"ip"`
	MAC          string  `json:"mac,omitempty"`
	Hostname     string  `json:"hostname"`
	DeviceType   string  `json:"device_type"` // "phone", "laptop", "tv", "desktop", "generic"
	DeviceIcon   string  `json:"device_icon"` // "📱", "💻", "📺", "🖥️", "🔌"
	RxMbps       float64 `json:"rx_mbps"`      // Client download rate (hotspot Tx to client)
	TxMbps       float64 `json:"tx_mbps"`      // Client upload rate (hotspot Rx from client)
	TotalMbps    float64 `json:"total_mbps"`
	TotalRxBytes int64   `json:"total_rx_bytes"`
	TotalTxBytes int64   `json:"total_tx_bytes"`
	Percent      float64 `json:"percent,omitempty"`
	IsActive     bool    `json:"is_active"`
	FirstSeen    string  `json:"first_seen,omitempty"`
	LastSeen     string  `json:"last_seen,omitempty"`
	Duration     string  `json:"duration,omitempty"`
}

// HotspotState holds the current status and metrics of the Mobile Hotspot.
type HotspotState struct {
	Active          bool            `json:"active"`
	InterfaceName   string          `json:"interface_name,omitempty"`
	GatewayIP       string          `json:"gateway_ip,omitempty"`
	Subnet          string          `json:"subnet,omitempty"`
	SSID            string          `json:"ssid,omitempty"`
	ClientCount     int             `json:"client_count"`
	TotalRxMbps     float64         `json:"total_rx_mbps"` // Total hotspot download rate (Internet -> Clients)
	TotalTxMbps     float64         `json:"total_tx_mbps"` // Total hotspot upload rate (Clients -> Internet)
	TotalMbps       float64         `json:"total_mbps"`
	TotalRxBytes    int64           `json:"total_rx_bytes"`
	TotalTxBytes    int64           `json:"total_tx_bytes"`
	BandwidthMethod string          `json:"bandwidth_method,omitempty"` // "interface_measured"
	Clients         []HotspotClient `json:"clients"`
	SampledAt       string          `json:"sampled_at"`
	Message         string          `json:"message,omitempty"`
}

// InferDeviceType infers device classification and emoji icon from hostname and MAC address.
func InferDeviceType(hostname, mac string) (deviceType, deviceIcon string) {
	h := strings.ToLower(hostname)
	switch {
	case strings.Contains(h, "iphone"), strings.Contains(h, "android"), strings.Contains(h, "redmi"),
		strings.Contains(h, "galaxy"), strings.Contains(h, "samsung"), strings.Contains(h, "pixel"),
		strings.Contains(h, "oneplus"), strings.Contains(h, "xiaomi"), strings.Contains(h, "oppo"),
		strings.Contains(h, "vivo"), strings.Contains(h, "realme"), strings.Contains(h, "phone"),
		strings.Contains(h, "mobile"), strings.Contains(h, "ipad"), strings.Contains(h, "tablet"),
		strings.Contains(h, "reddy"):
		return "phone", "📱"

	case strings.Contains(h, "laptop"), strings.Contains(h, "macbook"), strings.Contains(h, "thinkpad"),
		strings.Contains(h, "surface"), strings.Contains(h, "zenbook"), strings.Contains(h, "dell"),
		strings.Contains(h, "hp"), strings.Contains(h, "lenovo"), strings.Contains(h, "asus"),
		strings.Contains(h, "acer"), strings.Contains(h, "notebook"), strings.Contains(h, "book"):
		return "laptop", "💻"

	case strings.Contains(h, "tv"), strings.Contains(h, "smarttv"), strings.Contains(h, "bravia"),
		strings.Contains(h, "firetv"), strings.Contains(h, "chromecast"), strings.Contains(h, "roku"),
		strings.Contains(h, "appletv"), strings.Contains(h, "shield"):
		return "tv", "📺"

	case strings.Contains(h, "desktop"), strings.Contains(h, "pc"), strings.Contains(h, "tower"),
		strings.Contains(h, "workstation"), strings.Contains(h, "rig"):
		return "desktop", "🖥️"

	default:
		return "generic", "🔌"
	}
}

// CleanHostname cleans raw DNS or NetBIOS hostnames into human-readable labels.
func CleanHostname(raw, ip string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Unknown Device"
	}

	// Remove common local DNS suffixes (e.g. .mshome.net, .lan, .local, .localdomain)
	suffixes := []string{
		".mshome.net", ".mshome.net.",
		".home", ".home.",
		".lan", ".lan.",
		".local", ".local.",
		".localdomain", ".localdomain.",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(strings.ToLower(raw), s) {
			raw = raw[:len(raw)-len(s)]
		}
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Unknown Device"
	}

	// Replace dashes/underscores with clean spacing if appropriate
	clean := strings.ReplaceAll(raw, "-", " ")
	clean = strings.ReplaceAll(clean, "_", " ")
	parts := strings.Fields(clean)
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return raw
}

// CalculateHotspotClientThroughput computes Rx and Tx throughput in Mbps from byte deltas over monotonic elapsed seconds.
// Formula: (delta_bytes * 8) / elapsed_seconds / 1,000,000
func CalculateHotspotClientThroughput(rxBytesDelta, txBytesDelta uint64, elapsedSec float64) (rxMbps, txMbps, totalMbps float64) {
	if elapsedSec <= 0.0001 {
		return 0, 0, 0
	}

	rxMbps = (float64(rxBytesDelta) * 8.0) / elapsedSec / 1e6
	txMbps = (float64(txBytesDelta) * 8.0) / elapsedSec / 1e6
	totalMbps = rxMbps + txMbps

	rxMbps = math.Round(rxMbps*100) / 100
	txMbps = math.Round(txMbps*100) / 100
	totalMbps = math.Round(totalMbps*100) / 100

	return rxMbps, txMbps, totalMbps
}

// AggregateHotspotClients sorts clients descending by total bandwidth and computes percentage breakdown.
func AggregateHotspotClients(clients []HotspotClient, totalHotspotMbps float64) []HotspotClient {
	if len(clients) == 0 {
		return []HotspotClient{}
	}

	// Sort descending by total bandwidth, then by IP
	sort.Slice(clients, func(i, j int) bool {
		if clients[i].TotalMbps != clients[j].TotalMbps {
			return clients[i].TotalMbps > clients[j].TotalMbps
		}
		return clients[i].IP < clients[j].IP
	})

	var sumClientMbps float64
	for _, c := range clients {
		sumClientMbps += c.TotalMbps
	}

	refTotal := totalHotspotMbps
	if refTotal <= 0.001 {
		refTotal = sumClientMbps
	}

	for i := range clients {
		if refTotal > 0.001 {
			pct := (clients[i].TotalMbps / refTotal) * 100.0
			clients[i].Percent = math.Round(pct*10) / 10
			if clients[i].Percent > 100.0 {
				clients[i].Percent = 100.0
			}
		} else {
			clients[i].Percent = 0
		}
		// Activity determination: Active if throughput > 0.01 Mbps or recent active transfer
		if clients[i].TotalMbps > 0.01 || clients[i].RxMbps > 0.01 || clients[i].TxMbps > 0.01 {
			clients[i].IsActive = true
		}
	}

	return clients
}
