package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreFullFlow(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_netmon.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	now := time.Now()
	// Insert normal measurements
	err = s.SaveMeasurement(Measurement{
		Time:       now.Add(-10 * time.Minute).Format(time.RFC3339),
		NetworkID:  "test-net",
		Latency:    15.5,
		Jitter:     1.2,
		PacketLoss: 0,
		DNS:        12.0,
	})
	if err != nil {
		t.Fatalf("failed to save measurement: %v", err)
	}

	err = s.SaveMeasurement(Measurement{
		Time:       now.Add(-5 * time.Minute).Format(time.RFC3339),
		NetworkID:  "test-net",
		Latency:    25.5,
		Jitter:     2.4,
		PacketLoss: 0,
		DNS:        14.0,
	})
	if err != nil {
		t.Fatalf("failed to save measurement: %v", err)
	}

	// Insert consecutive outage measurements (100% loss, latency 0)
	err = s.SaveMeasurement(Measurement{
		Time:       now.Add(-2 * time.Minute).Format(time.RFC3339),
		NetworkID:  "test-net",
		Latency:    0,
		Jitter:     0,
		PacketLoss: 100,
		DNS:        0,
	})
	if err != nil {
		t.Fatalf("failed to save outage measurement 1: %v", err)
	}

	err = s.SaveMeasurement(Measurement{
		Time:       now.Add(-1 * time.Minute).Format(time.RFC3339),
		NetworkID:  "test-net",
		Latency:    0,
		Jitter:     0,
		PacketLoss: 100,
		DNS:        0,
	})
	if err != nil {
		t.Fatalf("failed to save outage measurement 2: %v", err)
	}

	// Insert speed test
	err = s.SaveSpeedTest(SpeedTest{
		Time:            now.Add(-3 * time.Minute).Format(time.RFC3339),
		DownloadMbps:    150.25,
		UploadMbps:      45.50,
		DurationSeconds: 2.1,
		BytesDown:       1000000,
		BytesUp:         1000000,
		Status:          "ok",
	})
	if err != nil {
		t.Fatalf("failed to save speed test: %v", err)
	}

	// Verify Summary calculation
	summary, err := s.GetSummary()
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}

	// Latency average should be (15.5 + 25.5) / 2 = 20.5 ms (excluding 0ms outage!)
	if summary.LatencyAvg != 20.5 {
		t.Fatalf("expected LatencyAvg 20.5, got %f", summary.LatencyAvg)
	}
	if summary.LatencyMin != 15.5 {
		t.Fatalf("expected LatencyMin 15.5, got %f", summary.LatencyMin)
	}
	if summary.LatencyMax != 25.5 {
		t.Fatalf("expected LatencyMax 25.5, got %f", summary.LatencyMax)
	}
	if summary.DownloadAvg != 150.25 {
		t.Fatalf("expected DownloadAvg 150.25, got %f", summary.DownloadAvg)
	}
	if summary.UploadAvg != 45.50 {
		t.Fatalf("expected UploadAvg 45.50, got %f", summary.UploadAvg)
	}
	if summary.Outages24h != 1 {
		t.Fatalf("expected 1 outage, got %d", summary.Outages24h)
	}

	// Verify History Window
	hist, err := s.GetHistoryWindow(60)
	if err != nil {
		t.Fatalf("failed to get history window: %v", err)
	}
	if len(hist) != 4 {
		t.Fatalf("expected 4 history measurements, got %d", len(hist))
	}

	// Verify Speed Tests Window
	stList, err := s.GetSpeedTestsWindow(60)
	if err != nil {
		t.Fatalf("failed to get speed tests: %v", err)
	}
	if len(stList) != 1 {
		t.Fatalf("expected 1 speed test, got %d", len(stList))
	}
	// Verify Latest Measurement
	latestMeas, err := s.GetLatestMeasurement()
	if err != nil {
		t.Fatalf("failed to get latest measurement: %v", err)
	}
	if latestMeas.PacketLoss != 100 {
		t.Fatalf("expected latest measurement to be outage (loss 100), got loss %f", latestMeas.PacketLoss)
	}
	if latestMeas.Timestamp == "" {
		t.Fatalf("expected non-empty Timestamp in latest measurement")
	}

	// Verify Ping Target upsert and query with LastChecked
	err = s.UpsertPingTarget(PingTarget{
		Host:    "google.com",
		IP:      "8.8.8.8",
		Latency: 12.3,
		Loss:    0,
		Status:  "up",
	})
	if err != nil {
		t.Fatalf("failed to upsert ping target: %v", err)
	}
	targets, err := s.GetPingTargets()
	if err != nil || len(targets) != 1 {
		t.Fatalf("failed to get ping targets: %v", err)
	}
	if targets[0].LastChecked == "" {
		t.Fatalf("expected non-empty LastChecked on ping target")
	}

	// Verify DNS Check upsert and query with LastChecked
	err = s.UpsertDNSCheck(DNSCheck{
		Host:     "google.com",
		TimeMs:   14.2,
		Resolver: "system",
	})
	if err != nil {
		t.Fatalf("failed to upsert dns check: %v", err)
	}
	checks, err := s.GetDNSChecks()
	if err != nil || len(checks) != 1 {
		t.Fatalf("failed to get dns checks: %v", err)
	}
	if checks[0].LastChecked == "" {
		t.Fatalf("expected non-empty LastChecked on dns check")
	}

	// Test Prune
	err = s.PruneOldRecords(1) // Keep 1 day
	if err != nil {
		t.Fatalf("failed to prune old records: %v", err)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_concurrent.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	const numWorkers = 20
	errChan := make(chan error, numWorkers*2)

	// Concurrent writers
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			mErr := s.SaveMeasurement(Measurement{
				Time:       time.Now().Format(time.RFC3339),
				NetworkID:  "concurrent-net",
				Latency:    float64(20 + id),
				PacketLoss: 0,
			})
			if mErr != nil {
				errChan <- mErr
				return
			}
			stErr := s.SaveSpeedTest(SpeedTest{
				Time:         time.Now().Format(time.RFC3339),
				DownloadMbps: float64(100 + id),
				UploadMbps:   float64(50 + id),
				Status:       "ok",
			})
			if stErr != nil {
				errChan <- stErr
				return
			}
			errChan <- nil
		}(i)
	}

	// Concurrent readers
	for i := 0; i < numWorkers; i++ {
		go func() {
			_, gErr := s.GetSummary()
			if gErr != nil {
				errChan <- gErr
				return
			}
			_, hErr := s.GetHistoryWindow(60)
			if hErr != nil {
				errChan <- hErr
				return
			}
			errChan <- nil
		}()
	}

	for i := 0; i < numWorkers*2; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("concurrent store operation failed: %v", err)
		}
	}
}

func TestStoreDownsamplingAndNulls(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_downsample.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	now := time.Now()
	// Insert measurements across 30 days
	for d := 0; d < 30; d++ {
		_ = s.SaveMeasurement(Measurement{
			Time:       now.Add(-time.Duration(d*24) * time.Hour).Format(time.RFC3339),
			NetworkID:  "hist-net",
			Latency:    15.0 + float64(d),
			PacketLoss: float64(d % 5),
		})
	}

	// Test 5m downsampling (> 24h = 1440m)
	hist5m, err := s.GetHistoryWindow(2880) // 2 days
	if err != nil {
		t.Fatalf("failed to query 5m downsampled history: %v", err)
	}
	if len(hist5m) == 0 {
		t.Fatalf("expected downsampled results for 2-day window")
	}

	// Test 1h downsampling (> 7 days = 10080m)
	hist1h, err := s.GetHistoryWindow(20000) // 14 days
	if err != nil {
		t.Fatalf("failed to query 1h downsampled history: %v", err)
	}
	if len(hist1h) == 0 {
		t.Fatalf("expected downsampled results for 14-day window")
	}

	// Verify Network Events logging and querying
	_ = s.LogNetworkEvent(NetworkEvent{
		Time:    now.Format(time.RFC3339),
		Network: "event-net",
		SSID:    "MyHomeWiFi",
		Gateway: "192.168.1.1",
	})

	events, err := s.GetRecentNetworkEvents(5)
	if err != nil || len(events) != 1 {
		t.Fatalf("expected 1 network event, got %d (err: %v)", len(events), err)
	}
	if events[0].SSID != "MyHomeWiFi" {
		t.Fatalf("expected SSID MyHomeWiFi, got %s", events[0].SSID)
	}
	if id := s.GetCurrentNetworkID(); id != "event-net" {
		t.Fatalf("expected current network id 'event-net', got '%s'", id)
	}
}
