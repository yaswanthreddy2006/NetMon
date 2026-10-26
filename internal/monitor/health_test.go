package monitor

import (
	"testing"
)

func TestCalculateHealthScenarios(t *testing.T) {
	weights := DefaultHealthWeights()

	// 1. Excellent Connection
	t.Run("Excellent Connection", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg:  12.0,
			PacketLoss:  0.0,
			JitterAvg:   1.5,
			DNSAvg:      10.0,
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 250.0,
		}
		eval := CalculateHealth(in, weights)
		if eval.Score < 95 {
			t.Fatalf("expected score >= 95 for excellent connection, got %d", eval.Score)
		}
		if eval.Rating != "GOOD" {
			t.Fatalf("expected rating GOOD, got %s", eval.Rating)
		}
		if len(eval.Reasons) > 0 {
			t.Fatalf("expected 0 issue reasons, got %v", eval.Reasons)
		}
	})

	// 2. Normal / Fair Connection
	t.Run("Normal Connection", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg:  45.0,
			PacketLoss:  0.5,
			JitterAvg:   6.0,
			DNSAvg:      35.0,
			Uptime24h:   99.5,
			Outages24h:  0,
			DownloadAvg: 60.0,
		}
		eval := CalculateHealth(in, weights)
		if eval.Score < 75 || eval.Score > 92 {
			t.Fatalf("expected score between 75 and 92 for normal connection, got %d", eval.Score)
		}
	})

	// 3. High Latency Connection
	t.Run("High Latency Connection", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg:  220.0, // High latency
			PacketLoss:  0.0,
			JitterAvg:   2.0,
			DNSAvg:      15.0,
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 100.0,
		}
		eval := CalculateHealth(in, weights)
		if eval.Components["latency"].Rating != "WARNING" && eval.Components["latency"].Rating != "POOR" {
			t.Fatalf("expected latency rating WARNING or POOR, got %s", eval.Components["latency"].Rating)
		}
		if eval.MainIssue != "Latency elevated to 220.0 ms" {
			t.Fatalf("expected main issue to be latency, got '%s'", eval.MainIssue)
		}
	})

	// 4. Packet Loss Connection
	t.Run("Packet Loss Connection", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg:  18.0,
			PacketLoss:  4.2, // Moderate packet loss
			JitterAvg:   2.0,
			DNSAvg:      12.0,
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 100.0,
		}
		eval := CalculateHealth(in, weights)
		if eval.Components["loss"].Score > 60 {
			t.Fatalf("expected packet loss score <= 60, got %f", eval.Components["loss"].Score)
		}
		if eval.MainIssue != "Packet loss elevated to 4.2%" {
			t.Fatalf("expected main issue to be packet loss, got '%s'", eval.MainIssue)
		}
	})

	// 5. Slow / Failing DNS
	t.Run("DNS Failure", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg:  15.0,
			PacketLoss:  0.0,
			JitterAvg:   1.0,
			DNSAvg:      180.0, // Extremely slow DNS
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 100.0,
		}
		eval := CalculateHealth(in, weights)
		if eval.Components["dns"].Rating != "POOR" {
			t.Fatalf("expected DNS rating POOR, got %s", eval.Components["dns"].Rating)
		}
		if eval.MainIssue != "Slow DNS resolution (180.0 ms)" {
			t.Fatalf("expected main issue to be DNS, got '%s'", eval.MainIssue)
		}
	})

	// 6. Internet Disconnected (100% Loss)
	t.Run("Internet Disconnected", func(t *testing.T) {
		in := HealthInput{
			LatencyAvg: 0.0,
			PacketLoss: 100.0,
			JitterAvg:  0.0,
			DNSAvg:     0.0,
			Uptime24h:  0.0,
			Outages24h: 5,
		}
		eval := CalculateHealth(in, weights)
		if eval.Score != 0 {
			t.Fatalf("expected 0 score for disconnected connection, got %d", eval.Score)
		}
		if eval.Rating != "POOR" {
			t.Fatalf("expected rating POOR, got %s", eval.Rating)
		}
	})

	// 7. Missing Data (e.g. no speed test and no DNS yet)
	t.Run("Missing Data Does Not Penalize User", func(t *testing.T) {
		inWithSpeed := HealthInput{
			LatencyAvg:  15.0,
			PacketLoss:  0.0,
			JitterAvg:   1.5,
			DNSAvg:      12.0,
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 200.0,
		}
		inWithoutSpeed := HealthInput{
			LatencyAvg:  15.0,
			PacketLoss:  0.0,
			JitterAvg:   1.5,
			DNSAvg:      12.0,
			Uptime24h:   100.0,
			Outages24h:  0,
			DownloadAvg: 0, // Unmeasured
		}

		evalWith := CalculateHealth(inWithSpeed, weights)
		evalWithout := CalculateHealth(inWithoutSpeed, weights)

		// Without speed test, score should still be near 100 and NOT penalized by 20 points!
		if evalWithout.Score < 95 {
			t.Fatalf("expected score >= 95 when unmeasured speed is omitted, got %d", evalWithout.Score)
		}
		if evalWithout.Score != evalWith.Score {
			// They should both be 100 or 99-100
			if evalWithout.Score < 98 {
				t.Fatalf("expected score ~100 without speed test, got %d", evalWithout.Score)
			}
		}
	})
}

func TestIndividualScoreFunctions(t *testing.T) {
	// 1. Loss Scoring
	s, ok := ScoreLoss(0.0)
	if !ok || s != 100 {
		t.Fatalf("expected 100 for 0 loss, got %f", s)
	}
	s, ok = ScoreLoss(0.5)
	if !ok || s < 85 || s > 100 {
		t.Fatalf("expected 85-100 for 0.5 loss, got %f", s)
	}
	s, ok = ScoreLoss(5.0)
	if !ok || s != 40 {
		t.Fatalf("expected 40 for 5.0 loss, got %f", s)
	}
	s, ok = ScoreLoss(10.0)
	if !ok || s != 0 {
		t.Fatalf("expected 0 for 10.0 loss, got %f", s)
	}
	s, ok = ScoreLoss(100.0)
	if !ok || s != 0 {
		t.Fatalf("expected 0 for 100 loss, got %f", s)
	}
	_, ok = ScoreLoss(-1.0)
	if ok {
		t.Fatalf("expected false for negative loss")
	}

	// 2. Latency Scoring
	s, ok = ScoreLatency(15.0)
	if !ok || s != 100 {
		t.Fatalf("expected 100 for 15ms latency, got %f", s)
	}
	s, ok = ScoreLatency(50.0)
	if !ok || s != 80 {
		t.Fatalf("expected 80 for 50ms latency, got %f", s)
	}
	s, ok = ScoreLatency(150.0)
	if !ok || s != 40 {
		t.Fatalf("expected 40 for 150ms latency, got %f", s)
	}
	s, ok = ScoreLatency(300.0)
	if !ok || s != 0 {
		t.Fatalf("expected 0 for 300ms latency, got %f", s)
	}
	_, ok = ScoreLatency(0.0)
	if ok {
		t.Fatalf("expected false for 0ms latency")
	}

	// 3. Jitter Scoring
	s, ok = ScoreJitter(2.0)
	if !ok || s != 100 {
		t.Fatalf("expected 100 for 2ms jitter, got %f", s)
	}
	s, ok = ScoreJitter(10.0)
	if !ok || s != 75 {
		t.Fatalf("expected 75 for 10ms jitter, got %f", s)
	}
	s, ok = ScoreJitter(30.0)
	if !ok || s != 30 {
		t.Fatalf("expected 30 for 30ms jitter, got %f", s)
	}

	// 4. DNS Scoring
	s, ok = ScoreDNS(10.0)
	if !ok || s != 100 {
		t.Fatalf("expected 100 for 10ms DNS, got %f", s)
	}
	s, ok = ScoreDNS(50.0)
	if !ok || s != 75 {
		t.Fatalf("expected 75 for 50ms DNS, got %f", s)
	}

	// 5. Uptime Scoring
	uptimeTests := []struct {
		uptime   float64
		outages  int
		wantMin  float64
		wantMax  float64
		expectOk bool
	}{
		{100.0, 0, 100, 100, true},
		{99.9, 0, 99, 100, true},
		{99.0, 0, 95, 95, true},
		{98.0, 0, 90, 92, true},
		{95.0, 0, 80, 80, true},
		{95.0, 1, 78, 80, true},
		{94.81, 0, 78, 80, true},
		{94.81, 10, 60, 66, true},
		{90.0, 0, 60, 60, true},
		{80.0, 0, 40, 40, true},
		{50.0, 0, 15, 15, true},
		{0.0, 0, 0, 0, true},
		{-1.0, 0, 0, 0, false},
	}

	var prevScore float64 = 101.0
	for _, ut := range uptimeTests {
		score, ok := ScoreUptime(ut.uptime, ut.outages)
		if ok != ut.expectOk {
			t.Fatalf("ScoreUptime(%v, %v): expected ok=%v, got %v", ut.uptime, ut.outages, ut.expectOk, ok)
		}
		if ok {
			if score < ut.wantMin || score > ut.wantMax {
				t.Fatalf("ScoreUptime(%v, %v): expected score in [%v, %v], got %f", ut.uptime, ut.outages, ut.wantMin, ut.wantMax, score)
			}
			if ut.outages == 0 {
				if score > prevScore {
					t.Fatalf("ScoreUptime monotonicity violated: uptime %v produced score %f > prevScore %f", ut.uptime, score, prevScore)
				}
				prevScore = score
			}
		}
	}

	// Specific regression test for 94.81% bug
	s94, ok94 := ScoreUptime(94.81, 0)
	if !ok94 || s94 <= 0 {
		t.Fatalf("ScoreUptime(94.81, 0) must be > 0, got %f (ok=%v)", s94, ok94)
	}
	s94Outages, ok94Outages := ScoreUptime(94.81, 10)
	if !ok94Outages || s94Outages <= 0 {
		t.Fatalf("ScoreUptime(94.81, 10) must be > 0, got %f (ok=%v)", s94Outages, ok94Outages)
	}

	// Verify health evaluation incorporates uptime score properly
	eval := CalculateHealth(HealthInput{
		LatencyAvg:  20.0,
		PacketLoss:  0.0,
		JitterAvg:   2.0,
		DNSAvg:      15.0,
		Uptime24h:   94.81,
		Outages24h:  10,
		DownloadAvg: 100.0,
	}, DefaultHealthWeights())
	if eval.Components["uptime"].Score <= 0 {
		t.Fatalf("expected uptime component score > 0 for 94.81%% uptime, got: %+v", eval.Components["uptime"])
	}
	if eval.Score < 80 {
		t.Fatalf("expected overall health score >= 80, got %d", eval.Score)
	}

	// 6. Speed Scoring
	s, ok = ScoreSpeed(100.0)
	if !ok || s != 100 {
		t.Fatalf("expected 100 for 100Mbps speed, got %f", s)
	}
	s, ok = ScoreSpeed(50.0)
	if !ok || s != 85 {
		t.Fatalf("expected 85 for 50Mbps speed, got %f", s)
	}
	s, ok = ScoreSpeed(25.0)
	if !ok || s != 70 {
		t.Fatalf("expected 70 for 25Mbps speed, got %f", s)
	}
	s, ok = ScoreSpeed(10.0)
	if !ok || s != 50 {
		t.Fatalf("expected 50 for 10Mbps speed, got %f", s)
	}

	// 7. RateScore
	if RateScore(90) != "GOOD" {
		t.Fatalf("expected GOOD for 90")
	}
	if RateScore(70) != "FAIR" {
		t.Fatalf("expected FAIR for 70")
	}
	if RateScore(50) != "WARNING" {
		t.Fatalf("expected WARNING for 50")
	}
	if RateScore(20) != "POOR" {
		t.Fatalf("expected POOR for 20")
	}
}
