package monitor

import (
	"math"
	"time"

	"netmon/internal/network"
	"netmon/internal/store"
)

// DiagnosticStatus represents overall connection operational status.
type DiagnosticStatus string

const (
	StatusOptimal  DiagnosticStatus = "optimal"
	StatusDegraded DiagnosticStatus = "degraded"
	StatusOffline  DiagnosticStatus = "offline"
	StatusUnknown  DiagnosticStatus = "unknown"
)

// LikelyCause identifies the classified origin of a network issue.
type LikelyCause string

const (
	CauseNone          LikelyCause = "none"
	CauseLocalNetwork  LikelyCause = "local_network"
	CauseISPUpstream   LikelyCause = "isp_upstream"
	CauseDNS           LikelyCause = "dns"
	CauseDisconnection LikelyCause = "disconnection"
	CauseTransition    LikelyCause = "network_transition"
	CauseSaturation    LikelyCause = "traffic_saturation"
	CauseInsufficient  LikelyCause = "insufficient_evidence"
)

// DiagnosisResult encapsulates the automated network diagnostic finding.
type DiagnosisResult struct {
	Status      DiagnosticStatus `json:"status"`
	LikelyCause LikelyCause      `json:"likely_cause"`
	Confidence  float64          `json:"confidence"` // 0.0 to 1.0
	Summary     string           `json:"summary"`
	Reason      string           `json:"reason"`
	EvaluatedAt string           `json:"evaluated_at"`
}

// DiagnosticInput aggregates measurements required for diagnosis.
type DiagnosticInput struct {
	Targets       []store.PingTarget
	DNSChecks     []store.DNSCheck
	Summary       store.Summary
	LiveTraffic   network.LiveTraffic
	RecentEvents  []store.NetworkEvent
	LastPingCycle time.Time
}

// Diagnose evaluates telemetry and isolates the most probable fault domain.
func Diagnose(in DiagnosticInput) DiagnosisResult {
	now := time.Now()
	res := DiagnosisResult{
		Status:      StatusOptimal,
		LikelyCause: CauseNone,
		Confidence:  0.90,
		Summary:     "Connection healthy",
		Reason:      "All local gateway, Internet, and DNS checks are within normal operating parameters.",
		EvaluatedAt: now.Format(time.RFC3339),
	}

	if len(in.Targets) == 0 && len(in.DNSChecks) == 0 {
		res.Status = StatusUnknown
		res.LikelyCause = CauseInsufficient
		res.Confidence = 0.30
		res.Summary = "Insufficient evidence"
		res.Reason = "No target ping or DNS checks recorded yet."
		return res
	}

	// 1. Separate Gateway target from Internet targets
	var gwTarget store.PingTarget
	var hasGW bool
	var internetTargets []store.PingTarget
	var totalLoss, totalLat float64
	var downCount, upCount int

	for _, t := range in.Targets {
		if t.Host == "gateway" {
			gwTarget = t
			hasGW = true
		} else {
			internetTargets = append(internetTargets, t)
			totalLoss += t.Loss
			if t.Latency > 0 {
				totalLat += t.Latency
				upCount++
			}
			if t.Loss >= 100 || t.Status == "down" {
				downCount++
			}
		}
	}

	avgNetLoss := 0.0
	if len(internetTargets) > 0 {
		avgNetLoss = totalLoss / float64(len(internetTargets))
	}
	avgNetLat := 0.0
	if upCount > 0 {
		avgNetLat = totalLat / float64(upCount)
	}

	// 2. Evaluate DNS checks
	var totalDnsTime float64
	var dnsCount int
	var dnsFailures int
	for _, d := range in.DNSChecks {
		if d.TimeMs > 0 {
			totalDnsTime += d.TimeMs
			dnsCount++
		} else {
			dnsFailures++
		}
	}
	avgDnsTime := 0.0
	if dnsCount > 0 {
		avgDnsTime = totalDnsTime / float64(dnsCount)
	}

	// 3. Rule D: Total Disconnection
	gwDown := hasGW && (gwTarget.Loss >= 100 || gwTarget.Status == "down")
	allInternetDown := len(internetTargets) > 0 && downCount == len(internetTargets)

	if allInternetDown {
		res.Status = StatusOffline
		res.LikelyCause = CauseDisconnection
		if gwDown {
			res.Confidence = 0.95
			res.Summary = "Likely local network disconnect"
			res.Reason = "Neither local gateway nor external Internet targets are reachable (100% loss)."
		} else {
			res.Confidence = 0.92
			res.Summary = "Likely total Internet outage"
			res.Reason = "Local gateway is responding, but 100% of Internet targets are unreachable."
		}
		return res
	}

	// 4. Rule E: Recent Network Interface / SSID Transition
	if len(in.RecentEvents) > 0 {
		// Events are ordered DESC by timestamp (index 0 is latest)
		lastEvent := in.RecentEvents[0]
		if evTime, err := time.Parse(time.RFC3339, lastEvent.Time); err == nil {
			if now.Sub(evTime) < 3*time.Minute && (in.Summary.PacketLoss > 2.0 || in.Summary.LatencyAvg > 80) {
				res.Status = StatusDegraded
				res.LikelyCause = CauseTransition
				res.Confidence = 0.78
				res.Summary = "Possible network transition issue"
				res.Reason = "Network adapter or SSID change detected within the last 3 minutes, coinciding with elevated packet loss/latency."
				return res
			}
		}
	}

	// 5. Rule A: Local Gateway & Wi-Fi Degradation
	gwDegraded := hasGW && (gwTarget.Loss > 2.0 || gwTarget.Latency > 25.0)
	if gwDegraded {
		if avgNetLoss > 2.0 || avgNetLat > 60.0 {
			res.Status = StatusDegraded
			res.LikelyCause = CauseLocalNetwork
			res.Confidence = 0.88
			res.Summary = "Likely local network / Wi-Fi issue"
			res.Reason = "Local router gateway is experiencing high latency or packet loss, propagating degradation to all external Internet traffic."
			return res
		}
	}

	// 6. Rule F: Traffic Saturation / Bufferbloat (High active I/O concurrent with latency/jitter rise)
	if in.LiveTraffic.RxMbps > 50.0 || in.LiveTraffic.TxMbps > 25.0 {
		if in.Summary.JitterAvg > 20.0 || in.Summary.LatencyAvg > 90.0 || avgNetLat > 90.0 {
			res.Status = StatusDegraded
			res.LikelyCause = CauseSaturation
			res.Confidence = 0.78
			res.Summary = "Possible local bandwidth saturation"
			res.Reason = "High concurrent live I/O throughput detected alongside increased jitter and latency variance."
			return res
		}
	}

	// 7. Rule B: Healthy Gateway, Degraded Internet Targets (ISP / Upstream Issue)
	gwHealthy := hasGW && gwTarget.Loss <= 1.0 && gwTarget.Latency <= 15.0
	internetDegraded := avgNetLoss > 2.0 || avgNetLat > 100.0 || in.Summary.PacketLoss > 2.0 || in.Summary.JitterAvg > 15.0

	if gwHealthy && internetDegraded {
		res.Status = StatusDegraded
		res.LikelyCause = CauseISPUpstream
		res.Confidence = 0.85
		res.Summary = "Likely ISP / upstream routing issue"
		res.Reason = "Local gateway connection is optimal, but external targets exhibit elevated latency, jitter, or packet loss."
		return res
	}

	// 8. Rule C: Healthy Ping, Failing / Slow DNS
	pingHealthy := (!hasGW || gwTarget.Loss == 0) && avgNetLoss < 1.0 && in.Summary.LatencyAvg < 70.0
	dnsDegraded := dnsFailures > 0 || avgDnsTime > 100.0

	if pingHealthy && dnsDegraded {
		res.Status = StatusDegraded
		res.LikelyCause = CauseDNS
		res.Confidence = 0.82
		res.Summary = "Likely DNS resolver issue"
		res.Reason = "ICMP ping latency to IP endpoints is normal, but DNS domain resolution is slow or failing."
		return res
	}

	// 9. Mild anomalies / Fallback
	if in.Summary.PacketLoss > 1.0 || in.Summary.LatencyAvg > 60.0 {
		res.Status = StatusDegraded
		res.LikelyCause = CauseInsufficient
		res.Confidence = 0.55
		res.Summary = "Possible minor network degradation"
		res.Reason = "Minor latency or loss detected without an isolated local gateway or DNS signature."
		return res
	}

	res.Confidence = math.Round(res.Confidence*100) / 100
	return res
}
