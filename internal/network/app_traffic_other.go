//go:build !windows

package network

import "time"

// SampleActiveAppTraffic is a cross-platform stub on non-Windows platforms.
func SampleActiveAppTraffic() ApplicationTraffic {
	return ApplicationTraffic{
		Timestamp:             time.Now().Format(time.RFC3339),
		SampleIntervalSeconds: 2.0,
		Applications:          []ProcessTraffic{},
	}
}
