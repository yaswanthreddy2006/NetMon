//go:build linux

package network

import (
	"os"
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

// GetConnectionInfo detects the active interface type and returns its details.
func GetConnectionInfo() (ConnectionInfo, bool) {
	iface := linuxActiveInterface()
	if iface == "" {
		return ConnectionInfo{
			Type:              "none",
			Status:            "disconnected",
			UnavailableReason: "No active default network interface found",
		}, false
	}
	if linuxIsWifi(iface) {
		return linuxWifiInfo(iface)
	}
	return linuxEthernetInfo(iface)
}

// linuxActiveInterface returns the interface used for the default route.
func linuxActiveInterface() string {
	if data, err := os.ReadFile("/proc/net/route"); err == nil {
		for _, line := range strings.Split(string(data), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "00000000" {
				return fields[0]
			}
		}
	}
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "ip", "route", "get", "1.1.1.1")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "dev" {
			return fields[i+1]
		}
	}
	return ""
}

func linuxIsWifi(iface string) bool {
	_, err := os.Stat("/sys/class/net/" + iface + "/wireless")
	return err == nil
}

func linuxWifiInfo(iface string) (ConnectionInfo, bool) {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "iw", "dev", iface, "link")
	if err != nil {
		return ConnectionInfo{
			Type:              "wifi",
			Status:            "unavailable",
			UnavailableReason: "iw command failed",
		}, false
	}
	if strings.Contains(string(out), "Not connected") {
		return ConnectionInfo{}, false
	}
	var info ConnectionInfo
	info.Type = "wifi"
	info.Status = "connected"
	info.Adapter = iface

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "SSID:"):
			info.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		case strings.HasPrefix(line, "Connected to "):
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				info.BSSID = fields[2]
			}
		case strings.HasPrefix(line, "signal:"):
			val := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, "signal:")), " dBm")
			info.RSSI, _ = strconv.Atoi(strings.TrimSpace(val))
			// Approximate quality
			if info.RSSI <= -100 {
				info.SignalQuality = 0
			} else if info.RSSI >= -50 {
				info.SignalQuality = 100
			} else {
				info.SignalQuality = 2 * (info.RSSI + 100)
			}
			info.SNR = info.SignalQuality / 2
		case strings.HasPrefix(line, "rx bitrate:"):
			if f := strings.Fields(line); len(f) >= 3 {
				mbps, err := strconv.ParseFloat(f[2], 64)
				if err == nil {
					info.LinkRate = int(mbps)
				}
			}
		case strings.HasPrefix(line, "freq:"):
			if f := strings.Fields(line); len(f) >= 2 {
				freq, _ := strconv.Atoi(f[1])
				switch {
				case freq >= 2401 && freq <= 2495:
					info.Band = "2.4GHz"
					info.Channel = (freq - 2407) / 5
				case freq >= 4900 && freq <= 5895:
					info.Band = "5GHz"
					info.Channel = (freq - 5000) / 5
				case freq >= 5925:
					info.Band = "6GHz"
					info.Channel = (freq - 5950) / 5
				}
			}
		}
	}
	if info.RSSI == 0 && info.LinkRate == 0 {
		return ConnectionInfo{}, false
	}
	return info, true
}

func linuxEthernetInfo(iface string) (ConnectionInfo, bool) {
	base := "/sys/class/net/" + iface + "/"
	speedData, err := os.ReadFile(base + "speed")
	if err != nil {
		return ConnectionInfo{}, false
	}
	speed, _ := strconv.Atoi(strings.TrimSpace(string(speedData)))
	if speed <= 0 {
		return ConnectionInfo{}, false
	}
	duplexData, _ := os.ReadFile(base + "duplex")
	duplex := strings.TrimSpace(string(duplexData))
	return ConnectionInfo{
		Type:     "ethernet",
		Adapter:  iface,
		LinkRate: speed,
		Duplex:   duplex,
		Status:   "connected",
	}, true
}
