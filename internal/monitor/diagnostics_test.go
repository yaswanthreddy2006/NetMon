package monitor

import (
	"testing"
	"time"

	"netmon/internal/network"
	"netmon/internal/store"
)

func TestDiagnoseScenarios(t *testing.T) {
	// Case A: Local Gateway & Wi-Fi Degradation
	t.Run("Case A: Local Gateway / Wi-Fi Issue", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 45.0, Loss: 8.0, Status: "up"},
				{Host: "google.com", IP: "8.8.8.8", Latency: 65.0, Loss: 8.0, Status: "up"},
			},
			DNSChecks: []store.DNSCheck{
				{Host: "google.com", TimeMs: 25.0},
			},
			Summary: store.Summary{
				LatencyAvg: 65.0,
				PacketLoss: 8.0,
			},
		}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseLocalNetwork {
			t.Fatalf("expected cause %s, got %s", CauseLocalNetwork, diag.LikelyCause)
		}
		if diag.Status != StatusDegraded {
			t.Fatalf("expected status degraded, got %s", diag.Status)
		}
	})

	// Case B: ISP / Upstream Issue (Gateway Healthy, Internet Targets Degraded)
	t.Run("Case B: ISP / Upstream Issue", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 2.1, Loss: 0.0, Status: "up"},
				{Host: "cloudflare.com", IP: "1.1.1.1", Latency: 140.0, Loss: 5.0, Status: "up"},
				{Host: "google.com", IP: "8.8.8.8", Latency: 155.0, Loss: 6.0, Status: "up"},
			},
			DNSChecks: []store.DNSCheck{
				{Host: "google.com", TimeMs: 20.0},
			},
			Summary: store.Summary{
				LatencyAvg: 147.5,
				PacketLoss: 5.5,
			},
		}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseISPUpstream {
			t.Fatalf("expected cause %s, got %s", CauseISPUpstream, diag.LikelyCause)
		}
		if diag.Status != StatusDegraded {
			t.Fatalf("expected status degraded, got %s", diag.Status)
		}
	})

	// Case C: DNS Resolution Failure / Degradation
	t.Run("Case C: DNS Issue", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 1.5, Loss: 0.0, Status: "up"},
				{Host: "1.1.1.1", IP: "1.1.1.1", Latency: 15.0, Loss: 0.0, Status: "up"},
			},
			DNSChecks: []store.DNSCheck{
				{Host: "google.com", TimeMs: 195.0}, // Very slow DNS
			},
			Summary: store.Summary{
				LatencyAvg: 15.0,
				PacketLoss: 0.0,
			},
		}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseDNS {
			t.Fatalf("expected cause %s, got %s", CauseDNS, diag.LikelyCause)
		}
	})

	// Case D: Total Disconnection
	t.Run("Case D: Total Internet Disconnection", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 0.0, Loss: 100.0, Status: "down"},
				{Host: "google.com", IP: "8.8.8.8", Latency: 0.0, Loss: 100.0, Status: "down"},
			},
			Summary: store.Summary{
				PacketLoss: 100.0,
			},
		}
		diag := Diagnose(in)
		if diag.Status != StatusOffline {
			t.Fatalf("expected status offline, got %s", diag.Status)
		}
		if diag.LikelyCause != CauseDisconnection {
			t.Fatalf("expected cause %s, got %s", CauseDisconnection, diag.LikelyCause)
		}
	})

	// Case E: Network Transition
	t.Run("Case E: Network Transition Issue", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 35.0, Loss: 4.0, Status: "up"},
			},
			Summary: store.Summary{
				LatencyAvg: 95.0,
				PacketLoss: 4.0,
			},
			RecentEvents: []store.NetworkEvent{
				{
					Time:    time.Now().Add(-30 * time.Second).Format(time.RFC3339),
					Network: "New_SSID_5G",
				},
			},
		}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseTransition {
			t.Fatalf("expected cause %s, got %s", CauseTransition, diag.LikelyCause)
		}
	})

	// Case F: Bandwidth Saturation
	t.Run("Case F: Live Traffic Saturation", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 8.0, Loss: 0.0, Status: "up"},
				{Host: "google.com", IP: "8.8.8.8", Latency: 110.0, Loss: 0.0, Status: "up"},
			},
			LiveTraffic: network.LiveTraffic{
				RxMbps: 92.5,
				TxMbps: 18.0,
			},
			Summary: store.Summary{
				LatencyAvg: 110.0,
				JitterAvg:  28.0,
			},
		}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseSaturation {
			t.Fatalf("expected cause %s, got %s", CauseSaturation, diag.LikelyCause)
		}
	})

	// Case G: Healthy Connection
	t.Run("Case G: Healthy Connection", func(t *testing.T) {
		in := DiagnosticInput{
			Targets: []store.PingTarget{
				{Host: "gateway", IP: "192.168.1.1", Latency: 1.8, Loss: 0.0, Status: "up"},
				{Host: "google.com", IP: "8.8.8.8", Latency: 14.0, Loss: 0.0, Status: "up"},
			},
			DNSChecks: []store.DNSCheck{
				{Host: "google.com", TimeMs: 12.0},
			},
			Summary: store.Summary{
				LatencyAvg: 14.0,
				PacketLoss: 0.0,
			},
		}
		diag := Diagnose(in)
		if diag.Status != StatusOptimal {
			t.Fatalf("expected status optimal, got %s", diag.Status)
		}
		if diag.LikelyCause != CauseNone {
			t.Fatalf("expected cause none, got %s", diag.LikelyCause)
		}
	})

	// Case H: Insufficient Data
	t.Run("Case H: Insufficient Data", func(t *testing.T) {
		in := DiagnosticInput{}
		diag := Diagnose(in)
		if diag.LikelyCause != CauseInsufficient {
			t.Fatalf("expected cause %s, got %s", CauseInsufficient, diag.LikelyCause)
		}
	})
}
