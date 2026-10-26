//go:build !darwin && !linux && !windows

package network

// ConnectionInfo describes the active network connection regardless of type.
type ConnectionInfo struct {
	Type              string `json:"type"`
	SSID              string `json:"ssid,omitempty"`
	BSSID             string `json:"bssid,omitempty"`
	RSSI              int    `json:"rssi"`
	SignalQuality     int    `json:"signal_quality"`
	Noise             int    `json:"noise"`
	SNR               int    `json:"snr"`
	Channel           int    `json:"channel"`
	ChannelWidth      string `json:"channel_width,omitempty"`
	Band              string `json:"band"`
	RadioType         string `json:"radio_type,omitempty"`
	LinkRate          int    `json:"link_rate"`
	Duplex            string `json:"duplex,omitempty"`
	Adapter           string `json:"adapter,omitempty"`
	Status            string `json:"status"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	IP                string `json:"ip,omitempty"`
	Gateway           string `json:"gateway,omitempty"`
}

// GetConnectionInfo is a no-op stub for unsupported platforms.
func GetConnectionInfo() (ConnectionInfo, bool) {
	return ConnectionInfo{
		Type:              "none",
		Status:            "unavailable",
		UnavailableReason: "Operating system not supported for native Wi-Fi/Ethernet metrics",
	}, false
}
