package network

import (
	"context"
	"time"
)

// PlatformSupport represents platform support level for a metric or feature.
type PlatformSupport string

const (
	SupportFull    PlatformSupport = "SUPPORTED"
	SupportPartial PlatformSupport = "PARTIALLY SUPPORTED"
	SupportNone    PlatformSupport = "NOT AVAILABLE"
)

// MetricQualityStatus represents explicit data quality states.
type MetricQualityStatus string

const (
	QualitySuccess          MetricQualityStatus = "SUCCESS"
	QualityFailed           MetricQualityStatus = "FAILED"
	QualityTimeout          MetricQualityStatus = "TIMEOUT"
	QualityUnavailable      MetricQualityStatus = "UNAVAILABLE"
	QualityPermissionDenied MetricQualityStatus = "PERMISSION_DENIED"
	QualityNxDomain         MetricQualityStatus = "NXDOMAIN"
	QualityServFail         MetricQualityStatus = "SERVFAIL"
)

// MeasurementRecord is a standardized wrapper for data quality tagging.
type MeasurementRecord struct {
	Timestamp time.Time           `json:"timestamp"`
	Metric    string              `json:"metric"`
	Value     any                 `json:"value"`
	Unit      string              `json:"unit"`
	Source    string              `json:"source"`
	Status    MetricQualityStatus `json:"status"`
	Error     string              `json:"error,omitempty"`
}

// ConnectivityProvider abstracts network gateway and latency probing.
type ConnectivityProvider interface {
	DefaultGateway() string
	DetectNetwork() Info
}

// DNSProvider abstracts DNS server detection and DNS query benchmarking.
type DNSProvider interface {
	DetectConfiguredServers() []string
	MeasureLookup(ctx context.Context, host, resolver string) (time.Duration, MetricQualityStatus, error)
}

// WifiProvider abstracts OS-level Wi-Fi and Ethernet adapter telemetry.
type WifiProvider interface {
	GetConnectionInfo() (ConnectionInfo, bool)
}

// ThroughputProvider abstracts network bandwidth measurement.
type ThroughputProvider interface {
	MeasureDownload(ctx context.Context, targetURL string, streams int) (mbps float64, totalBytes int64, durSec float64, err error)
	MeasureUpload(ctx context.Context, targetURL string, sizeBytes int64, streams int) (mbps float64, totalBytes int64, durSec float64, err error)
}

// HotspotProvider abstracts mobile hotspot state and connected client discovery.
type HotspotProvider interface {
	SampleHotspotStatus() HotspotState
}

// PlatformMatrix documents capability availability across supported platforms.
type PlatformMatrix struct {
	Platform       string          `json:"platform"`
	GatewayDetect  PlatformSupport `json:"gateway_detect"`
	ICMPPing       PlatformSupport `json:"icmp_ping"`
	TCPFallback    PlatformSupport `json:"tcp_fallback"`
	DNSDetection   PlatformSupport `json:"dns_detection"`
	WifiRSSI       PlatformSupport `json:"wifi_rssi"`
	WifiPHYRate    PlatformSupport `json:"wifi_phy_rate"`
	WifiBandChan   PlatformSupport `json:"wifi_band_channel"`
	ThroughputTest PlatformSupport `json:"throughput_test"`
	HotspotAdapter PlatformSupport `json:"hotspot_adapter"`
	HotspotClients PlatformSupport `json:"hotspot_clients"`
	HotspotPerIP   PlatformSupport `json:"hotspot_per_ip_bandwidth"`
}

// GetPlatformCapabilities returns the matrix for the current runtime platform.
func GetPlatformCapabilities(goos string) PlatformMatrix {
	switch goos {
	case "windows":
		return PlatformMatrix{
			Platform:       "Windows (10/11/Server)",
			GatewayDetect:  SupportFull,
			ICMPPing:       SupportFull,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportFull,
			WifiRSSI:       SupportFull,
			WifiPHYRate:    SupportFull,
			WifiBandChan:   SupportFull,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportFull,
			HotspotClients: SupportFull,
			HotspotPerIP:   SupportPartial, // Interface-level accurate; per-IP requires packet filter driver
		}
	case "linux":
		return PlatformMatrix{
			Platform:       "Linux (kernel / iproute2 / wireless-tools)",
			GatewayDetect:  SupportFull,
			ICMPPing:       SupportFull,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportFull,
			WifiRSSI:       SupportFull,
			WifiPHYRate:    SupportFull,
			WifiBandChan:   SupportFull,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportFull,
			HotspotClients: SupportFull,
			HotspotPerIP:   SupportPartial,
		}
	case "darwin":
		return PlatformMatrix{
			Platform:       "macOS (Darwin)",
			GatewayDetect:  SupportFull,
			ICMPPing:       SupportFull,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportFull,
			WifiRSSI:       SupportFull,
			WifiPHYRate:    SupportFull,
			WifiBandChan:   SupportFull,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportPartial,
			HotspotClients: SupportPartial,
			HotspotPerIP:   SupportNone,
		}
	case "android":
		return PlatformMatrix{
			Platform:       "Android (via termux/NDK)",
			GatewayDetect:  SupportFull,
			ICMPPing:       SupportFull,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportPartial,
			WifiRSSI:       SupportPartial,
			WifiPHYRate:    SupportPartial,
			WifiBandChan:   SupportPartial,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportPartial,
			HotspotClients: SupportPartial,
			HotspotPerIP:   SupportNone,
		}
	case "ios":
		return PlatformMatrix{
			Platform:       "iOS (Sandboxed)",
			GatewayDetect:  SupportPartial,
			ICMPPing:       SupportPartial,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportPartial,
			WifiRSSI:       SupportNone, // Sandboxed by Apple
			WifiPHYRate:    SupportNone,
			WifiBandChan:   SupportNone,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportNone,
			HotspotClients: SupportNone,
			HotspotPerIP:   SupportNone,
		}
	default:
		return PlatformMatrix{
			Platform:       goos,
			GatewayDetect:  SupportPartial,
			ICMPPing:       SupportPartial,
			TCPFallback:    SupportFull,
			DNSDetection:   SupportPartial,
			WifiRSSI:       SupportNone,
			WifiPHYRate:    SupportNone,
			WifiBandChan:   SupportNone,
			ThroughputTest: SupportFull,
			HotspotAdapter: SupportNone,
			HotspotClients: SupportNone,
			HotspotPerIP:   SupportNone,
		}
	}
}
