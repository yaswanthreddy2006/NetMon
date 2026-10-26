package network

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ProcessTraffic represents real-time bandwidth consumption for a specific application/process.
type ProcessTraffic struct {
	PID         uint32  `json:"pid"`
	Name        string  `json:"name"`           // Friendly application name (e.g., "Google Chrome")
	Process     string  `json:"process"`        // Raw executable filename (e.g., "chrome.exe")
	Path        string  `json:"path,omitempty"` // Optional executable path
	RxMbps      float64 `json:"rx_mbps"`
	TxMbps      float64 `json:"tx_mbps"`
	TotalMbps   float64 `json:"total_mbps"`
	RxBytes     int64   `json:"rx_bytes"`
	TxBytes     int64   `json:"tx_bytes"`
	Connections int     `json:"connections"`
	Percent     float64 `json:"percent,omitempty"`
}

// ApplicationTraffic holds a sampled snapshot of all application network bandwidth.
type ApplicationTraffic struct {
	Timestamp             string           `json:"timestamp"`
	SampleIntervalSeconds float64          `json:"sample_interval_seconds"`
	TotalRxMbps           float64          `json:"total_rx_mbps"`
	TotalTxMbps           float64          `json:"total_tx_mbps"`
	TotalMbps             float64          `json:"total_mbps"`
	Applications          []ProcessTraffic `json:"applications"`
	UnattributedRxMbps    float64          `json:"unattributed_rx_mbps,omitempty"`
	UnattributedTxMbps    float64          `json:"unattributed_tx_mbps,omitempty"`
}

// FriendlyProcessName maps raw executable names to clean, readable application labels.
func FriendlyProcessName(execName string) string {
	base := strings.ToLower(filepath.Base(execName))
	switch base {
	case "chrome.exe", "chrome":
		return "Google Chrome"
	case "msedge.exe", "msedgewebview2.exe", "msedge":
		return "Microsoft Edge"
	case "brave.exe", "brave":
		return "Brave Browser"
	case "firefox.exe", "firefox":
		return "Mozilla Firefox"
	case "code.exe", "code":
		return "VS Code"
	case "discord.exe", "discord":
		return "Discord"
	case "slack.exe", "slack":
		return "Slack"
	case "spotify.exe", "spotify":
		return "Spotify"
	case "steam.exe", "steamwebhelper.exe", "steam":
		return "Steam"
	case "telegram.exe", "telegram":
		return "Telegram"
	case "zoom.exe", "zoom":
		return "Zoom"
	case "teams.exe", "teams":
		return "Microsoft Teams"
	case "netmon_server.exe", "netmon.exe", "netmon":
		return "NetMon Server"
	case "antigravity ide.exe", "antigravity.exe", "antigravity":
		return "Antigravity IDE"
	case "language_server_windows_x64.exe", "gopls.exe", "gopls":
		return "Go Language Server"
	case "node.exe", "node":
		return "Node.js"
	case "python.exe", "python3.exe", "python":
		return "Python"
	case "curl.exe", "curl":
		return "cURL"
	case "git.exe", "git":
		return "Git"
	case "powershell.exe", "pwsh.exe", "powershell", "pwsh":
		return "PowerShell"
	case "cmd.exe":
		return "Command Prompt"
	case "explorer.exe":
		return "Windows Explorer"
	case "svchost.exe":
		return "Windows Services (svchost)"
	case "system":
		return "System Kernel"
	}

	clean := strings.TrimSuffix(filepath.Base(execName), ".exe")
	if len(clean) > 0 {
		return strings.ToUpper(clean[:1]) + clean[1:]
	}
	return execName
}

// CalculateProcessThroughput computes Rx/Tx Mbps from byte deltas over monotonic elapsed seconds.
func CalculateProcessThroughput(rxBytesDelta, txBytesDelta uint64, elapsedSec float64) (rxMbps, txMbps, totalMbps float64) {
	if elapsedSec <= 0.001 {
		return 0, 0, 0
	}
	rxMbps = (float64(rxBytesDelta) * 8.0) / elapsedSec / 1e6
	txMbps = (float64(txBytesDelta) * 8.0) / elapsedSec / 1e6
	totalMbps = rxMbps + txMbps
	return rxMbps, txMbps, totalMbps
}

// AggregateAndSortAppTraffic aggregates raw process records, calculates percentages, and returns top N items.
func AggregateAndSortAppTraffic(procs []ProcessTraffic, maxTop int, sampleInterval float64) ApplicationTraffic {
	if maxTop <= 0 {
		maxTop = 10
	}

	var totalRx, totalTx, totalAll float64

	// Filter out non-zero activity and sum
	active := make([]ProcessTraffic, 0, len(procs))
	for _, p := range procs {
		p.TotalMbps = p.RxMbps + p.TxMbps
		if p.TotalMbps > 0.001 || p.RxBytes > 0 || p.TxBytes > 0 {
			totalRx += p.RxMbps
			totalTx += p.TxMbps
			totalAll += p.TotalMbps
			active = append(active, p)
		}
	}

	// Sort descending by total bandwidth
	sort.Slice(active, func(i, j int) bool {
		return active[i].TotalMbps > active[j].TotalMbps
	})

	// Calculate percentage of total process traffic
	for i := range active {
		if totalAll > 0 {
			active[i].Percent = (active[i].TotalMbps / totalAll) * 100.0
		}
	}

	var topApps []ProcessTraffic
	if len(active) > maxTop {
		topApps = active[:maxTop]
		var otherRx, otherTx, otherTotal float64
		var otherRxBytes, otherTxBytes int64
		var otherConns int
		for _, o := range active[maxTop:] {
			otherRx += o.RxMbps
			otherTx += o.TxMbps
			otherTotal += o.TotalMbps
			otherRxBytes += o.RxBytes
			otherTxBytes += o.TxBytes
			otherConns += o.Connections
		}
		if otherTotal > 0.001 {
			var otherPct float64
			if totalAll > 0 {
				otherPct = (otherTotal / totalAll) * 100.0
			}
			topApps = append(topApps, ProcessTraffic{
				PID:         0,
				Name:        "Other Applications",
				Process:     "other",
				RxMbps:      round2(otherRx),
				TxMbps:      round2(otherTx),
				TotalMbps:   round2(otherTotal),
				RxBytes:     otherRxBytes,
				TxBytes:     otherTxBytes,
				Connections: otherConns,
				Percent:     round1(otherPct),
			})
		}
	} else {
		topApps = active
	}

	// Round values for presentation
	for i := range topApps {
		topApps[i].RxMbps = round2(topApps[i].RxMbps)
		topApps[i].TxMbps = round2(topApps[i].TxMbps)
		topApps[i].TotalMbps = round2(topApps[i].TotalMbps)
		topApps[i].Percent = round1(topApps[i].Percent)
	}

	return ApplicationTraffic{
		Timestamp:             time.Now().Format(time.RFC3339),
		SampleIntervalSeconds: sampleInterval,
		TotalRxMbps:           round2(totalRx),
		TotalTxMbps:           round2(totalTx),
		TotalMbps:             round2(totalAll),
		Applications:          topApps,
	}
}
