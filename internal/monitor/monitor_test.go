package monitor

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"netmon/internal/store"
)

func TestMonitorDynamicIntervals(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_monitor.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := Config{
		PingTargets:   []string{"127.0.0.1"},
		DNSTargets:    []string{"127.0.0.1"},
		PingInterval:  60 * time.Second,
		SpeedInterval: 30 * time.Minute,
		PingCount:     1,
		DownloadURL:   "http://127.0.0.1/down",
		UploadURL:     "http://127.0.0.1/up",
	}

	mon := New(cfg, s, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon.Start(ctx)

	// Update interval dynamically to 100ms
	newCfg := cfg
	newCfg.PingInterval = 100 * time.Millisecond
	newCfg.SpeedInterval = 200 * time.Millisecond
	mon.SetConfig(newCfg)

	cur := mon.GetConfig()
	if cur.PingInterval != 100*time.Millisecond {
		t.Fatalf("expected ping interval 100ms, got %v", cur.PingInterval)
	}
	if cur.SpeedInterval != 200*time.Millisecond {
		t.Fatalf("expected speed interval 200ms, got %v", cur.SpeedInterval)
	}

	// Allow workers to cycle with new interval
	time.Sleep(300 * time.Millisecond)

	// Cancel context and verify clean shutdown
	cancel()
	mon.Wait()
}

func TestValidateConfig(t *testing.T) {
	// Test invalid negative and zero parameters
	invalidCfg := Config{
		PingTargets:    []string{"", "  "},
		DNSTargets:     []string{" "},
		PingInterval:   -10 * time.Second,
		SpeedInterval:  -5 * time.Minute,
		PingCount:      -3,
		DownloadURL:    "",
		UploadURL:      "",
		SpeedSizeBytes: -100,
		SpeedStreams:   -1,
	}

	validated := ValidateConfig(invalidCfg)

	if len(validated.PingTargets) == 0 {
		t.Fatalf("expected default ping targets for empty inputs")
	}
	if len(validated.DNSTargets) == 0 {
		t.Fatalf("expected default dns targets for empty inputs")
	}
	if validated.PingInterval <= 0 {
		t.Fatalf("expected positive ping interval, got %v", validated.PingInterval)
	}
	if validated.SpeedInterval <= 0 {
		t.Fatalf("expected positive speed interval, got %v", validated.SpeedInterval)
	}
	if validated.PingCount < 1 || validated.PingCount > 20 {
		t.Fatalf("expected ping count between 1 and 20, got %d", validated.PingCount)
	}
	if validated.DownloadURL == "" || validated.UploadURL == "" {
		t.Fatalf("expected non-empty fallback URLs")
	}
	if validated.SpeedSizeBytes <= 0 || validated.SpeedStreams <= 0 {
		t.Fatalf("expected positive speed size and streams")
	}
}

func TestBuildDNSQuery(t *testing.T) {
	query := buildDNSQuery("google.com", 0x1234)
	if len(query) < 12 {
		t.Fatalf("expected at least 12 bytes DNS query header, got %d", len(query))
	}
	// Verify transaction ID
	if query[0] != 0x12 || query[1] != 0x34 {
		t.Fatalf("expected tx ID 0x1234, got 0x%02x%02x", query[0], query[1])
	}
}

func TestOutageMeasurementHandling(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_outage.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := Config{
		PingTargets:   []string{"127.0.0.1"},
		DNSTargets:    []string{"127.0.0.1"},
		PingInterval:  60 * time.Second,
		SpeedInterval: 30 * time.Minute,
		PingCount:     1,
	}

	mon := New(cfg, s, logger)

	// Simulate outage cycle
	mon.runPingCycle(context.Background())

	meas, err := s.GetLatestMeasurement()
	if err != nil {
		t.Fatalf("failed to get latest measurement: %v", err)
	}
	if meas.Timestamp == "" {
		t.Fatalf("expected non-empty measurement timestamp")
	}

	// Verify that speed cycle handles unreachable targets gracefully without panic
	mon.runSpeedCycle(context.Background())

	st, err := s.GetLatestSpeedTest()
	if err != nil {
		t.Fatalf("failed to get latest speed test: %v", err)
	}
	// Unreachable URLs should have marked status != ok and not saved 0 as successful
	if st.Status == "ok" && st.DownloadMbps == 0 {
		t.Fatalf("should not store 0 mbps with status 'ok'")
	}
}

func TestMonitorConcurrentWorkersAndCancellation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_workers.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := Config{
		PingTargets:   []string{"127.0.0.1"},
		DNSTargets:    []string{"127.0.0.1"},
		PingInterval:  50 * time.Millisecond,
		SpeedInterval: 50 * time.Millisecond,
		PingCount:     1,
		DownloadURL:   "http://127.0.0.1:9999/down",
		UploadURL:     "http://127.0.0.1:9999/up",
	}

	mon := New(cfg, s, logger)
	ctx, cancel := context.WithCancel(context.Background())

	// Start all 4 workers
	mon.Start(ctx)

	// Dynamically update intervals repeatedly under load
	for i := 0; i < 5; i++ {
		mon.SetConfig(Config{
			PingTargets:   []string{"127.0.0.1"},
			DNSTargets:    []string{"127.0.0.1"},
			PingInterval:  time.Duration(60+i*10) * time.Millisecond,
			SpeedInterval: time.Duration(80+i*10) * time.Millisecond,
			PingCount:     1,
		})
		time.Sleep(30 * time.Millisecond)
	}

	// Cancel and ensure clean, non-blocking termination
	cancel()

	done := make(chan struct{})
	go func() {
		mon.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded
	case <-time.After(5 * time.Second):
		t.Fatalf("monitor workers did not stop within 5 seconds of context cancellation")
	}
}

func TestConfigFileLoadingAndOverrides(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Non-existent file should return default config without error
	cfg, err := LoadConfigFile(filepath.Join(tempDir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected nil error for missing config file, got %v", err)
	}
	if len(cfg.PingTargets) == 0 {
		t.Fatalf("expected default ping targets")
	}

	// 2. Corrupt / invalid JSON file should return error
	corruptPath := filepath.Join(tempDir, "corrupt.json")
	_ = os.WriteFile(corruptPath, []byte("invalid json {{"), 0644)
	_, err = LoadConfigFile(corruptPath)
	if err == nil {
		t.Fatalf("expected error for corrupt JSON, got nil")
	}

	// 3. Valid JSON with partial fields
	validPath := filepath.Join(tempDir, "custom.json")
	validJSON := `{
		"ping_targets": ["1.1.1.1", "8.8.8.8"],
		"ping_interval_s": 15,
		"speed_interval_m": 10,
		"ping_count": 3
	}`
	_ = os.WriteFile(validPath, []byte(validJSON), 0644)
	customCfg, err := LoadConfigFile(validPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customCfg.PingInterval != 15*time.Second {
		t.Fatalf("expected 15s ping interval, got %v", customCfg.PingInterval)
	}
	if customCfg.SpeedInterval != 10*time.Minute {
		t.Fatalf("expected 10m speed interval, got %v", customCfg.SpeedInterval)
	}
	if customCfg.PingCount != 3 {
		t.Fatalf("expected ping count 3, got %d", customCfg.PingCount)
	}
}
