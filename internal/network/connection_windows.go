//go:build windows

package network

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// ConnectionInfo describes the active network connection regardless of type.
type ConnectionInfo struct {
	Type              string `json:"type"` // "wifi", "ethernet", or ""
	SSID              string `json:"ssid,omitempty"`
	BSSID             string `json:"bssid,omitempty"`
	RSSI              int    `json:"rssi"`               // signal strength dBm (negative; 0 = unavailable)
	SignalQuality     int    `json:"signal_quality"`      // 0-100 percentage
	Noise             int    `json:"noise"`              // noise floor dBm (negative; 0 = unavailable)
	SNR               int    `json:"snr"`                // signal-to-noise ratio dB (0 = unavailable)
	Channel           int    `json:"channel"`            // WiFi channel number (0 = unavailable)
	ChannelWidth      string `json:"channel_width,omitempty"`
	Band              string `json:"band"`               // "2.4GHz", "5GHz", "6GHz", or ""
	RadioType         string `json:"radio_type,omitempty"`
	LinkRate          int    `json:"link_rate"`          // Wi-Fi PHY Link Rate / Ethernet Link Speed Mbps (0 = unavailable)
	Duplex            string `json:"duplex,omitempty"`   // "full", "half", or ""
	Adapter           string `json:"adapter,omitempty"`
	Status            string `json:"status"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	IP                string `json:"ip,omitempty"`
	Gateway           string `json:"gateway,omitempty"`
}

// GetConnectionInfo detects the active Windows interface and returns its connection details.
func GetConnectionInfo() (ConnectionInfo, bool) {
	// Try Wi-Fi first via netsh wlan show interfaces
	if info, ok := windowsWifiInfo(); ok {
		return info, true
	}

	// Fall back to Ethernet
	if info, ok := windowsEthernetInfo(); ok {
		return info, true
	}

	return ConnectionInfo{
		Type:              "none",
		Status:            "disconnected",
		UnavailableReason: "No active Wi-Fi or Ethernet connection detected",
	}, false
}

func windowsWifiInfo() (ConnectionInfo, bool) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "wlan", "show", "interfaces")
	if err != nil {
		return ConnectionInfo{
			Type:              "wifi",
			Status:            "unavailable",
			UnavailableReason: fmt.Sprintf("netsh command failed: %v", err),
		}, false
	}

	text := string(out)
	if strings.Contains(text, "There is no wireless interface") || strings.Contains(text, "no wireless interface") {
		return ConnectionInfo{}, false
	}

	// Scan lines and detect connected interface block
	scanner := bufio.NewScanner(strings.NewReader(text))
	var explicitRSSI int
	var hasExplicitRSSI bool
	var info ConnectionInfo
	info.Type = "wifi"
	info.Status = "connected"

	var isConnectedInterface bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch {
		case key == "state" || key == "status":
			lowerVal := strings.ToLower(val)
			if lowerVal == "connected" || strings.Contains(lowerVal, "verbunden") || strings.Contains(lowerVal, "connecté") {
				isConnectedInterface = true
			}
		case key == "name" || strings.Contains(key, "schnittstellenname"):
			// If not yet confirmed connected, reset adapter fields
			if !isConnectedInterface {
				info.Adapter = ""
				info.SSID = ""
				info.BSSID = ""
			}
		case key == "description" || strings.Contains(key, "beschreibung"):
			info.Adapter = val
		case key == "ssid":
			info.SSID = val
		case key == "bssid" || key == "ap bssid":
			info.BSSID = val
		case key == "radio type" || key == "funktyp":
			info.RadioType = val
		case key == "band":
			// Windows 11 outputs "5 GHz", "2.4 GHz", "6 GHz"
			cleanBand := strings.ReplaceAll(val, " ", "")
			if cleanBand == "2GHz" {
				cleanBand = "2.4GHz"
			}
			info.Band = cleanBand
		case key == "channel" || key == "kanal" || key == "canal":
			if ch, err := strconv.Atoi(val); err == nil {
				info.Channel = ch
			}
		case key == "signal":
			// "Signal : 99%"
			sigStr := strings.TrimSuffix(val, "%")
			if pct, err := strconv.Atoi(strings.TrimSpace(sigStr)); err == nil {
				info.SignalQuality = pct
			}
		case key == "rssi":
			// "Rssi : -47"
			rssiClean := strings.TrimSpace(strings.TrimSuffix(val, "dBm"))
			if r, err := strconv.Atoi(rssiClean); err == nil && r < 0 {
				explicitRSSI = r
				hasExplicitRSSI = true
			}
		case key == "receive rate (mbps)" || key == "transmit rate (mbps)" || strings.Contains(key, "rate (mbps)"):
			if r, err := strconv.ParseFloat(val, 64); err == nil && int(r) > info.LinkRate {
				info.LinkRate = int(r)
			}
		}
	}

	// RSSI determination
	if hasExplicitRSSI {
		info.RSSI = explicitRSSI
	} else if info.SignalQuality > 0 {
		info.RSSI = (info.SignalQuality / 2) - 100
	}

	if info.SignalQuality > 0 {
		info.SNR = info.SignalQuality / 2
	}

	// Channel to Band deduction if not explicitly provided
	if info.Band == "" && info.Channel > 0 {
		switch {
		case info.Channel <= 14:
			info.Band = "2.4GHz"
		case info.Channel >= 32 && info.Channel <= 177:
			info.Band = "5GHz"
		case info.Channel >= 180:
			info.Band = "6GHz"
		}
	}

	info.IP = OutboundIP()
	info.Gateway = DefaultGateway()
	if info.Band == "" && info.Channel > 0 {
		switch {
		case info.Channel <= 14:
			info.Band = "2.4GHz"
		case info.Channel >= 32 && info.Channel <= 177:
			info.Band = "5GHz"
		case info.Channel >= 180:
			info.Band = "6GHz"
		}
	}

	// Channel width inference from radio type and rate
	if info.RadioType != "" {
		switch {
		case strings.Contains(info.RadioType, "802.11ax") || strings.Contains(info.RadioType, "802.11be"):
			if info.LinkRate >= 1200 {
				info.ChannelWidth = "80/160 MHz"
			} else if info.LinkRate >= 574 {
				info.ChannelWidth = "40/80 MHz"
			} else {
				info.ChannelWidth = "20/40 MHz"
			}
		case strings.Contains(info.RadioType, "802.11ac"):
			if info.LinkRate >= 866 {
				info.ChannelWidth = "80 MHz"
			} else if info.LinkRate >= 400 {
				info.ChannelWidth = "40 MHz"
			} else {
				info.ChannelWidth = "20 MHz"
			}
		case strings.Contains(info.RadioType, "802.11n"):
			if info.LinkRate >= 300 {
				info.ChannelWidth = "40 MHz"
			} else {
				info.ChannelWidth = "20 MHz"
			}
		}
	}

	if info.RSSI != 0 || info.LinkRate > 0 || info.Channel > 0 || info.SSID != "" {
		return info, true
	}

	return ConnectionInfo{}, false
}

func windowsEthernetInfo() (ConnectionInfo, bool) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "interface", "ipv4", "show", "subinterfaces")
	if err != nil {
		return ConnectionInfo{}, false
	}

	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			state := fields[1]
			if state == "1" || state == "connected" {
				name := strings.Join(fields[4:], " ")
				lower := strings.ToLower(name)
				if (strings.Contains(lower, "ethernet") || strings.Contains(lower, "lan") || strings.Contains(lower, "eth")) &&
					!strings.Contains(lower, "virtual") && !strings.Contains(lower, "vethernet") && !strings.Contains(lower, "loopback") {
					return ConnectionInfo{
						Type:     "ethernet",
						Adapter:  name,
						LinkRate: 1000, // standard Gigabit Ethernet link rate
						Duplex:   "full",
						Status:   "connected",
						IP:       OutboundIP(),
						Gateway:  DefaultGateway(),
					}, true
				}
			}
		}
	}

	return ConnectionInfo{}, false
}
