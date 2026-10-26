//go:build !windows

package network

import (
	"time"
)

// SampleActiveHotspotStatus provides fallback hotspot sampling on non-Windows platforms.
func SampleActiveHotspotStatus() HotspotState {
	return HotspotState{
		Active:      false,
		ClientCount: 0,
		Clients:     []HotspotClient{},
		SampledAt:   time.Now().Format(time.RFC3339),
		Message:     "Mobile Hotspot monitoring is only active when Wi-Fi sharing is enabled on this OS.",
	}
}
