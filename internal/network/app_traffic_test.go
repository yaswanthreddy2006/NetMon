package network

import (
	"math"
	"testing"
	"time"
)

func TestCalculateProcessThroughput(t *testing.T) {
	tests := []struct {
		name          string
		rxBytesDelta  uint64
		txBytesDelta  uint64
		elapsedSec    float64
		wantRxMbps    float64
		wantTxMbps    float64
		wantTotalMbps float64
	}{
		{
			name:          "Zero duration should return 0",
			rxBytesDelta:  1000000,
			txBytesDelta:  500000,
			elapsedSec:    0,
			wantRxMbps:    0,
			wantTxMbps:    0,
			wantTotalMbps: 0,
		},
		{
			name:          "Sub-millisecond duration should return 0",
			rxBytesDelta:  1000000,
			txBytesDelta:  500000,
			elapsedSec:    0.0005,
			wantRxMbps:    0,
			wantTxMbps:    0,
			wantTotalMbps: 0,
		},
		{
			name:          "Zero bytes delta",
			rxBytesDelta:  0,
			txBytesDelta:  0,
			elapsedSec:    2.0,
			wantRxMbps:    0,
			wantTxMbps:    0,
			wantTotalMbps: 0,
		},
		{
			name:          "1MB RX in 1 second = 8.0 Mbps",
			rxBytesDelta:  1000000,
			txBytesDelta:  250000,
			elapsedSec:    1.0,
			wantRxMbps:    8.0,
			wantTxMbps:    2.0,
			wantTotalMbps: 10.0,
		},
		{
			name:          "2MB RX in 2 seconds = 8.0 Mbps",
			rxBytesDelta:  2000000,
			txBytesDelta:  1000000,
			elapsedSec:    2.0,
			wantRxMbps:    8.0,
			wantTxMbps:    4.0,
			wantTotalMbps: 12.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rxMbps, txMbps, totalMbps := CalculateProcessThroughput(tt.rxBytesDelta, tt.txBytesDelta, tt.elapsedSec)
			if math.Abs(rxMbps-tt.wantRxMbps) > 0.01 {
				t.Errorf("rxMbps = %v, want %v", rxMbps, tt.wantRxMbps)
			}
			if math.Abs(txMbps-tt.wantTxMbps) > 0.01 {
				t.Errorf("txMbps = %v, want %v", txMbps, tt.wantTxMbps)
			}
			if math.Abs(totalMbps-tt.wantTotalMbps) > 0.01 {
				t.Errorf("totalMbps = %v, want %v", totalMbps, tt.wantTotalMbps)
			}
		})
	}
}

func TestFriendlyProcessName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"chrome.exe", "Google Chrome"},
		{"msedge.exe", "Microsoft Edge"},
		{"brave.exe", "Brave Browser"},
		{"firefox.exe", "Mozilla Firefox"},
		{"Code.exe", "VS Code"},
		{"discord.exe", "Discord"},
		{"slack.exe", "Slack"},
		{"spotify.exe", "Spotify"},
		{"steam.exe", "Steam"},
		{"netmon_server.exe", "NetMon Server"},
		{"antigravity.exe", "Antigravity IDE"},
		{"node.exe", "Node.js"},
		{"python.exe", "Python"},
		{"powershell.exe", "PowerShell"},
		{"custom_app.exe", "Custom_app"},
		{"system", "System Kernel"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FriendlyProcessName(tt.input)
			if got != tt.want {
				t.Errorf("FriendlyProcessName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAggregateAndSortAppTraffic(t *testing.T) {
	t.Run("Zero traffic returns empty list with 0 totals", func(t *testing.T) {
		result := AggregateAndSortAppTraffic(nil, 8, 2.0)
		if len(result.Applications) != 0 {
			t.Errorf("expected 0 apps, got %d", len(result.Applications))
		}
		if result.TotalMbps != 0 {
			t.Errorf("expected 0 total Mbps, got %v", result.TotalMbps)
		}
	})

	t.Run("Sorts descending and calculates percentage correctly", func(t *testing.T) {
		procs := []ProcessTraffic{
			{PID: 100, Name: "VS Code", Process: "code.exe", RxMbps: 1.0, TxMbps: 1.0, TotalMbps: 2.0},
			{PID: 200, Name: "Google Chrome", Process: "chrome.exe", RxMbps: 7.0, TxMbps: 1.0, TotalMbps: 8.0},
		}

		res := AggregateAndSortAppTraffic(procs, 5, 2.0)
		if len(res.Applications) != 2 {
			t.Fatalf("expected 2 apps, got %d", len(res.Applications))
		}

		// First should be Chrome (8.0 Mbps total)
		if res.Applications[0].Name != "Google Chrome" {
			t.Errorf("first app should be Chrome, got %s", res.Applications[0].Name)
		}
		if res.Applications[0].TotalMbps != 8.0 {
			t.Errorf("Chrome totalMbps = %v, want 8.0", res.Applications[0].TotalMbps)
		}
		if res.Applications[0].Percent != 80.0 {
			t.Errorf("Chrome percent = %v, want 80.0", res.Applications[0].Percent)
		}

		// Second should be VS Code (2.0 Mbps total, 20%)
		if res.Applications[1].Name != "VS Code" {
			t.Errorf("second app should be VS Code, got %s", res.Applications[1].Name)
		}
		if res.Applications[1].Percent != 20.0 {
			t.Errorf("VS Code percent = %v, want 20.0", res.Applications[1].Percent)
		}

		if res.TotalMbps != 10.0 {
			t.Errorf("res.TotalMbps = %v, want 10.0", res.TotalMbps)
		}
	})

	t.Run("Groups into Other Applications when count exceeds maxTop", func(t *testing.T) {
		procs := make([]ProcessTraffic, 10)
		for i := 0; i < 10; i++ {
			procs[i] = ProcessTraffic{
				PID:       uint32(1000 + i),
				Name:      "App",
				Process:   "app.exe",
				RxMbps:    1.0,
				TxMbps:    0.0,
				TotalMbps: 1.0,
			}
		}

		res := AggregateAndSortAppTraffic(procs, 3, 2.0)
		// 3 top apps + 1 'Other Applications' = 4
		if len(res.Applications) != 4 {
			t.Fatalf("expected 4 entries (3 top + other), got %d", len(res.Applications))
		}

		last := res.Applications[3]
		if last.Name != "Other Applications" {
			t.Errorf("last item should be Other Applications, got %s", last.Name)
		}
		if last.TotalMbps != 7.0 {
			t.Errorf("Other Applications totalMbps = %v, want 7.0", last.TotalMbps)
		}
		if last.Percent != 70.0 {
			t.Errorf("Other Applications percent = %v, want 70.0", last.Percent)
		}
	})
}

func TestPIDReuseHandling(t *testing.T) {
	tracker := &WindowsAppTrafficTracker{
		prevSnapshot: make(map[uint32]procSnapshot),
		lastSampled:  time.Now().Add(-2 * time.Second),
	}

	// Initial process with PID 5000 and CreationTime 10000
	tracker.prevSnapshot[5000] = procSnapshot{
		PID:          5000,
		Name:         "old_app.exe",
		CreationTime: 10000,
		RxBytes:      5000000,
		TxBytes:      1000000,
	}

	// Case 1: Same process continuing
	currSame := procSnapshot{
		PID:          5000,
		Name:         "old_app.exe",
		CreationTime: 10000,
		RxBytes:      6000000,
		TxBytes:      1200000,
	}
	prev := tracker.prevSnapshot[5000]
	if prev.CreationTime == currSame.CreationTime {
		rxDelta := currSame.RxBytes - prev.RxBytes
		if rxDelta != 1000000 {
			t.Errorf("expected rxDelta 1000000, got %d", rxDelta)
		}
	} else {
		t.Errorf("creation time should match for same process")
	}

	// Case 2: Process killed, PID reused by new app with CreationTime 20000
	currReused := procSnapshot{
		PID:          5000,
		Name:         "new_app.exe",
		CreationTime: 20000,
		RxBytes:      10000, // Starts fresh with lower bytes
		TxBytes:      5000,
	}
	if prev.CreationTime == currReused.CreationTime {
		t.Errorf("PID reuse should be detected because creation times differ")
	}
}
