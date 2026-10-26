//go:build darwin

package network

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
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

// GetConnectionInfo detects the active interface and returns its connection details.
func GetConnectionInfo() (ConnectionInfo, bool) {
	_, activeIface := darwinRoute()

	// Try WiFi via system_profiler
	if spOut, err := ExecCommandWithTimeout(5*time.Second, "system_profiler", "SPAirPortDataType", "-json"); err == nil {
		if info, ok := parseMacWifi(spOut, activeIface); ok {
			return info, true
		}
	}

	// Fall back to Ethernet
	if activeIface != "" {
		return macEthernetInfo(activeIface)
	}
	return ConnectionInfo{
		Type:              "none",
		Status:            "disconnected",
		UnavailableReason: "No active Wi-Fi or Ethernet interface detected",
	}, false
}

type spAirPortData struct {
	SPAirPortDataType []struct {
		Interfaces []struct {
			Name    string `json:"_name"`
			Current *struct {
				SSID        string `json:"spairport_network_name"`
				BSSID       string `json:"spairport_network_bssid"`
				Channel     string `json:"spairport_network_channel"`
				Rate        int    `json:"spairport_network_rate"`
				SignalNoise string `json:"spairport_signal_noise"`
			} `json:"spairport_current_network_information"`
		} `json:"spairport_airport_interfaces"`
	} `json:"SPAirPortDataType"`
}

func parseMacWifi(data []byte, activeIface string) (ConnectionInfo, bool) {
	var sp spAirPortData
	if err := json.Unmarshal(data, &sp); err != nil {
		return ConnectionInfo{}, false
	}
	for _, section := range sp.SPAirPortDataType {
		for _, iface := range section.Interfaces {
			if activeIface != "" && iface.Name != activeIface {
				continue
			}
			cur := iface.Current
			if cur == nil || cur.SignalNoise == "" {
				continue
			}
			var info ConnectionInfo
			info.Type = "wifi"
			info.Status = "connected"
			info.Adapter = iface.Name
			info.SSID = cur.SSID
			info.BSSID = cur.BSSID

			parts := strings.SplitN(cur.SignalNoise, " / ", 2)
			if len(parts) == 2 {
				rssi, e1 := strconv.Atoi(strings.TrimSuffix(parts[0], " dBm"))
				noise, e2 := strconv.Atoi(strings.TrimSuffix(parts[1], " dBm"))
				if e1 == nil && e2 == nil {
					info.RSSI = rssi
					info.Noise = noise
					info.SNR = rssi - noise
					if rssi <= -100 {
						info.SignalQuality = 0
					} else if rssi >= -50 {
						info.SignalQuality = 100
					} else {
						info.SignalQuality = 2 * (rssi + 100)
					}
				}
			}

			if chParts := strings.SplitN(cur.Channel, " ", 2); len(chParts) >= 1 {
				info.Channel, _ = strconv.Atoi(chParts[0])
			}
			switch {
			case strings.Contains(cur.Channel, "6GHz"):
				info.Band = "6GHz"
			case strings.Contains(cur.Channel, "5GHz"):
				info.Band = "5GHz"
			case strings.Contains(cur.Channel, "2GHz"):
				info.Band = "2.4GHz"
			}
			if strings.Contains(cur.Channel, "MHz") {
				fields := strings.Fields(cur.Channel)
				for _, f := range fields {
					if strings.HasSuffix(f, "MHz") || strings.HasSuffix(f, "MHz)") {
						info.ChannelWidth = strings.Trim(f, "()")
					}
				}
			}
			info.LinkRate = cur.Rate
			return info, info.RSSI != 0 || info.LinkRate != 0
		}
	}
	return ConnectionInfo{}, false
}

func macEthernetInfo(iface string) (ConnectionInfo, bool) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "ifconfig", iface)
	if err != nil {
		return ConnectionInfo{}, false
	}
	info := parseIfconfigMedia(string(out))
	if info.LinkRate == 0 {
		return ConnectionInfo{}, false
	}
	info.Type = "ethernet"
	info.Adapter = iface
	info.Status = "connected"
	return info, true
}

func parseIfconfigMedia(output string) ConnectionInfo {
	var info ConnectionInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "media:") {
			continue
		}
		start := strings.Index(line, "(")
		end := strings.LastIndex(line, ")")
		if start < 0 || end <= start {
			break
		}
		inner := line[start+1 : end]
		fields := strings.Fields(inner)
		if len(fields) == 0 {
			break
		}
		speedStr := fields[0]
		nonDigit := strings.IndexFunc(speedStr, func(r rune) bool { return r < '0' || r > '9' })
		if nonDigit > 0 {
			info.LinkRate, _ = strconv.Atoi(speedStr[:nonDigit])
		}
		for _, f := range fields[1:] {
			switch f {
			case "<full-duplex>":
				info.Duplex = "full"
			case "<half-duplex>":
				info.Duplex = "half"
			}
		}
		break
	}
	return info
}
