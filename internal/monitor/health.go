package monitor

import (
	"fmt"
	"math"
	"sort"
)

// HealthWeights defines configurable component weights for the Connection Health Score.
type HealthWeights struct {
	Loss    float64 `json:"loss"`
	Latency float64 `json:"latency"`
	Jitter  float64 `json:"jitter"`
	DNS     float64 `json:"dns"`
	Uptime  float64 `json:"uptime"`
	Speed   float64 `json:"speed"`
}

// DefaultHealthWeights returns production default weights.
func DefaultHealthWeights() HealthWeights {
	return HealthWeights{
		Loss:    0.30,
		Latency: 0.25,
		Jitter:  0.15,
		DNS:     0.15,
		Uptime:  0.15,
		Speed:   0.10,
	}
}

// ComponentScore represents the evaluation of an individual metric.
type ComponentScore struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Raw    string  `json:"raw"`
	Rating string  `json:"rating"` // GOOD, FAIR, WARNING, POOR
}

// HealthEvaluation holds the overall score, component breakdown, and diagnostic reasoning.
type HealthEvaluation struct {
	Score      int                       `json:"score"`
	Rating     string                    `json:"rating"`
	Components map[string]ComponentScore `json:"components"`
	Reasons    []string                  `json:"reasons"`
	MainIssue  string                    `json:"main_issue"`
}

// ScoreLoss normalizes packet loss (%) to a 0-100 score.
func ScoreLoss(loss float64) (float64, bool) {
	if loss < 0 || math.IsNaN(loss) {
		return 0, false
	}
	if loss == 0 {
		return 100, true
	}
	if loss <= 1.0 {
		return math.Round(100 - (loss/1.0)*15), true // 100 -> 85
	}
	if loss <= 5.0 {
		return math.Round(85 - ((loss-1.0)/4.0)*45), true // 85 -> 40
	}
	if loss <= 10.0 {
		return math.Round(40 - ((loss-5.0)/5.0)*40), true // 40 -> 0
	}
	return 0, true
}

// ScoreLatency normalizes latency (ms) to a 0-100 score.
func ScoreLatency(lat float64) (float64, bool) {
	if lat <= 0 || math.IsNaN(lat) {
		return 0, false
	}
	if lat <= 20 {
		return 100, true
	}
	if lat <= 50 {
		return math.Round(100 - ((lat-20)/30)*20), true // 100 -> 80
	}
	if lat <= 150 {
		return math.Round(80 - ((lat-50)/100)*40), true // 80 -> 40
	}
	if lat <= 300 {
		return math.Round(40 - ((lat-150)/150)*40), true // 40 -> 0
	}
	return 0, true
}

// ScoreJitter normalizes jitter (ms) to a 0-100 score.
func ScoreJitter(jit float64) (float64, bool) {
	if jit < 0 || math.IsNaN(jit) {
		return 0, false
	}
	if jit <= 3 {
		return 100, true
	}
	if jit <= 10 {
		return math.Round(100 - ((jit-3)/7)*25), true // 100 -> 75
	}
	if jit <= 30 {
		return math.Round(75 - ((jit-10)/20)*45), true // 75 -> 30
	}
	return math.Max(0, math.Round(30-(jit-30))), true
}

// ScoreDNS normalizes DNS resolution time (ms) to a 0-100 score.
func ScoreDNS(dns float64) (float64, bool) {
	if dns <= 0 || math.IsNaN(dns) {
		return 0, false
	}
	if dns <= 15 {
		return 100, true
	}
	if dns <= 50 {
		return math.Round(100 - ((dns-15)/35)*25), true // 100 -> 75
	}
	if dns <= 150 {
		return math.Round(75 - ((dns-50)/100)*45), true // 75 -> 30
	}
	return math.Max(0, math.Round(30-(dns-150))), true
}

// ScoreUptime normalizes uptime percentage (0-100) and outage count to a continuous 0-100 score.
// Returns (0, false) for missing data (uptimePct < 0 or NaN).
func ScoreUptime(uptimePct float64, outages int) (float64, bool) {
	if math.IsNaN(uptimePct) || uptimePct < 0 {
		if outages > 0 {
			penalty := math.Min(30.0, float64(outages)*3.0)
			return math.Max(0, math.Round(100-penalty)), true
		}
		return 0, false
	}

	var base float64
	switch {
	case uptimePct >= 100:
		base = 100
	case uptimePct >= 99.0:
		base = 95.0 + ((uptimePct-99.0)/1.0)*5.0 // 99% -> 95, 100% -> 100
	case uptimePct >= 95.0:
		base = 80.0 + ((uptimePct-95.0)/4.0)*15.0 // 95% -> 80, 99% -> 95
	case uptimePct >= 90.0:
		base = 60.0 + ((uptimePct-90.0)/5.0)*20.0 // 90% -> 60, 95% -> 80
	case uptimePct >= 80.0:
		base = 40.0 + ((uptimePct-80.0)/10.0)*20.0 // 80% -> 40, 90% -> 60
	case uptimePct >= 50.0:
		base = 15.0 + ((uptimePct-50.0)/30.0)*25.0 // 50% -> 15, 80% -> 40
	case uptimePct > 0:
		base = (uptimePct / 50.0) * 15.0 // 0% -> 0, 50% -> 15
	default:
		base = 0
	}

	if outages > 0 {
		penalty := math.Min(15.0, float64(outages)*1.5)
		base = math.Max(0, base-penalty)
	}

	return math.Max(0, math.Min(100, math.Round(base))), true
}

// ScoreSpeed normalizes download bandwidth (Mbps) to a 0-100 score.
func ScoreSpeed(dlMbps float64) (float64, bool) {
	if dlMbps <= 0 || math.IsNaN(dlMbps) {
		return 0, false
	}
	if dlMbps >= 100 {
		return 100, true
	}
	if dlMbps >= 50 {
		return math.Round(85 + ((dlMbps-50)/50)*15), true
	}
	if dlMbps >= 25 {
		return math.Round(70 + ((dlMbps-25)/25)*15), true
	}
	if dlMbps >= 10 {
		return math.Round(50 + ((dlMbps-10)/15)*20), true
	}
	return math.Max(0, math.Round(dlMbps*5)), true
}

// RateScore maps 0-100 score to rating level.
func RateScore(score float64) string {
	if score >= 85 {
		return "GOOD"
	}
	if score >= 65 {
		return "FAIR"
	}
	if score >= 40 {
		return "WARNING"
	}
	return "POOR"
}

// HealthInput represents telemetry metrics passed into the health calculator.
type HealthInput struct {
	LatencyAvg  float64
	PacketLoss  float64
	JitterAvg   float64
	DNSAvg      float64
	Uptime24h   float64
	Outages24h  int
	DownloadAvg float64
}

// CalculateHealth computes the overall Connection Health score and diagnostics.
func CalculateHealth(in HealthInput, weights HealthWeights) HealthEvaluation {
	// If 100% loss (total disconnect)
	if in.PacketLoss >= 100 {
		return HealthEvaluation{
			Score:  0,
			Rating: "POOR",
			Components: map[string]ComponentScore{
				"loss": {Name: "Packet Loss", Score: 0, Raw: "100%", Rating: "POOR"},
			},
			Reasons:   []string{"Connection offline: 100% packet loss"},
			MainIssue: "Connection offline: 100% packet loss",
		}
	}

	components := make(map[string]ComponentScore)
	type issue struct {
		score float64
		text  string
	}
	var issues []issue

	var weightedSum, totalWeight float64

	// Helper to add scored component
	add := func(key, name, raw string, w float64, scoreVal float64, ok bool, issueText string) {
		if !ok {
			return
		}
		weightedSum += scoreVal * w
		totalWeight += w
		r := RateScore(scoreVal)
		components[key] = ComponentScore{
			Name:   name,
			Score:  scoreVal,
			Raw:    raw,
			Rating: r,
		}
		if scoreVal < 85 {
			issues = append(issues, issue{score: scoreVal, text: issueText})
		}
	}

	if s, ok := ScoreLoss(in.PacketLoss); ok {
		add("loss", "Packet Loss", fmt.Sprintf("%.1f%%", in.PacketLoss), weights.Loss, s, ok, fmt.Sprintf("Packet loss elevated to %.1f%%", in.PacketLoss))
	}
	if s, ok := ScoreLatency(in.LatencyAvg); ok {
		add("latency", "Latency", fmt.Sprintf("%.1f ms", in.LatencyAvg), weights.Latency, s, ok, fmt.Sprintf("Latency elevated to %.1f ms", in.LatencyAvg))
	}
	if s, ok := ScoreJitter(in.JitterAvg); ok {
		add("jitter", "Jitter", fmt.Sprintf("%.1f ms", in.JitterAvg), weights.Jitter, s, ok, fmt.Sprintf("Jitter elevated to %.1f ms", in.JitterAvg))
	}
	if s, ok := ScoreDNS(in.DNSAvg); ok {
		add("dns", "DNS", fmt.Sprintf("%.1f ms", in.DNSAvg), weights.DNS, s, ok, fmt.Sprintf("Slow DNS resolution (%.1f ms)", in.DNSAvg))
	}
	if s, ok := ScoreUptime(in.Uptime24h, in.Outages24h); ok {
		add("uptime", "Uptime", fmt.Sprintf("%.1f%%", in.Uptime24h), weights.Uptime, s, ok, fmt.Sprintf("Uptime degraded (%.1f%%, %d outages)", in.Uptime24h, in.Outages24h))
	}
	if s, ok := ScoreSpeed(in.DownloadAvg); ok {
		add("speed", "Speed", fmt.Sprintf("%.1f Mbps", in.DownloadAvg), weights.Speed, s, ok, fmt.Sprintf("Download speed limited (%.1f Mbps)", in.DownloadAvg))
	}

	finalScore := 0
	if totalWeight > 0 {
		finalScore = int(math.Round(weightedSum / totalWeight))
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].score < issues[j].score
	})

	var reasons []string
	for _, iss := range issues {
		reasons = append(reasons, iss.text)
	}

	mainIssue := "All network metrics within optimal thresholds"
	if len(reasons) > 0 {
		mainIssue = reasons[0]
	}

	return HealthEvaluation{
		Score:      finalScore,
		Rating:     RateScore(float64(finalScore)),
		Components: components,
		Reasons:    reasons,
		MainIssue:  mainIssue,
	}
}
