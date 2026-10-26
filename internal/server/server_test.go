package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"netmon/internal/monitor"
	"netmon/internal/store"
)

func TestServerEndpoints(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_server.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := monitor.DefaultConfig()
	mon := monitor.New(cfg, s, logger)

	h := NewHandler(context.Background(), s, mon, "v1.0.0")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/data", h.GetData)
	mux.HandleFunc("GET /api/latest", h.GetData)
	mux.HandleFunc("GET /api/config", h.GetConfig)
	mux.HandleFunc("POST /api/config", h.SaveConfig)
	mux.HandleFunc("GET /api/version", h.GetVersion)

	// Test GET /api/config
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/config, got %d", w.Code)
	}

	var cp ConfigPayload
	if err := json.NewDecoder(w.Body).Decode(&cp); err != nil {
		t.Fatalf("failed to decode config payload: %v", err)
	}
	if len(cp.PingTargets) == 0 {
		t.Fatalf("expected non-empty ping targets")
	}

	// Test POST /api/config with valid payload
	validPayload := ConfigPayload{
		PingTargets:    []string{"1.1.1.1", "8.8.8.8"},
		DNSTargets:     []string{"1.1.1.1"},
		PingIntervalS:  15,
		SpeedIntervalM: 10,
		PingCount:      4,
	}
	body, _ := json.Marshal(validPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from POST /api/config, got %d", w.Code)
	}

	// Verify updated config in monitor
	curCfg := mon.GetConfig()
	if curCfg.PingCount != 4 {
		t.Fatalf("expected ping count 4, got %d", curCfg.PingCount)
	}

	// Test POST /api/config with invalid payload (e.g. ping_interval < 10)
	invalidPayload := validPayload
	invalidPayload.PingIntervalS = 2
	body, _ = json.Marshal(invalidPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from invalid POST /api/config, got %d", w.Code)
	}

	// Insert a test speed test record
	_ = s.SaveSpeedTest(store.SpeedTest{
		Time:         time.Now().Format(time.RFC3339),
		DownloadMbps: 250.5,
		UploadMbps:   120.0,
		Status:       "ok",
	})

	// Insert measurement and ping target
	_ = s.SaveMeasurement(store.Measurement{
		Time:       time.Now().Format(time.RFC3339),
		NetworkID:  "test-net",
		Latency:    18.5,
		Jitter:     2.1,
		PacketLoss: 0,
		DNS:        15.0,
	})
	_ = s.UpsertPingTarget(store.PingTarget{
		Host:    "gateway",
		IP:      "192.168.1.1",
		Latency: 2.5,
		Loss:    0,
		Status:  "up",
	})

	// Test GET /api/data
	req = httptest.NewRequest(http.MethodGet, "/api/data?minutes=60", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/data, got %d", w.Code)
	}

	var data DashboardData
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode dashboard data: %v", err)
	}
	if data.ServerTime == "" {
		t.Fatalf("expected non-empty ServerTime in dashboard data")
	}
	if data.LatestMeasurement == nil || data.LatestMeasurement.Latency != 18.5 {
		t.Fatalf("expected LatestMeasurement in dashboard data, got %+v", data.LatestMeasurement)
	}
	if len(data.Targets) == 0 || data.Targets[0].LastChecked == "" {
		t.Fatalf("expected targets with LastChecked timestamp, got %+v", data.Targets)
	}
	if data.Diagnosis.Status == "" || data.Diagnosis.LikelyCause == "" {
		t.Fatalf("expected populated Diagnosis in dashboard data, got %+v", data.Diagnosis)
	}
	if data.SpeedTest == nil || data.SpeedTest.DownloadMbps != 250.5 {
		t.Fatalf("expected speed test in data response, got %+v", data.SpeedTest)
	}
	if len(data.SpeedTests) == 0 {
		t.Fatalf("expected speed_tests list in data response")
	}

	// Test GET /api/version
	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/version, got %d", w.Code)
	}

	// Test GET /api/latest
	req = httptest.NewRequest(http.MethodGet, "/api/latest", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GET /api/latest, got %d", w.Code)
	}
	var latestData DashboardData
	if err := json.NewDecoder(w.Body).Decode(&latestData); err != nil {
		t.Fatalf("failed to decode /api/latest: %v", err)
	}
	if latestData.ServerTime == "" {
		t.Fatalf("expected valid ServerTime in /api/latest")
	}
}

func TestServerEmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty_server.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := monitor.New(monitor.DefaultConfig(), s, logger)
	h := NewHandler(context.Background(), s, mon, "v1.0.0")

	// GET /api/data on freshly created empty DB should succeed with clean defaults
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	h.GetData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on empty DB, got %d", w.Code)
	}

	var data DashboardData
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if data.LatestMeasurement != nil {
		t.Fatalf("expected nil LatestMeasurement for empty DB, got %+v", data.LatestMeasurement)
	}
	if data.SpeedTest != nil {
		t.Fatalf("expected nil SpeedTest for empty DB, got %+v", data.SpeedTest)
	}
	if data.Diagnosis.Status == "" {
		t.Fatalf("expected non-empty Diagnosis status for empty DB")
	}
}

func TestServerSaveConfigValidationEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "config_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := monitor.New(monitor.DefaultConfig(), s, logger)
	h := NewHandler(context.Background(), s, mon, "v1.0.0")

	// 1. Oversized payload (> 64 KB)
	largeBody := bytes.Repeat([]byte("A"), 70*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(largeBody))
	w := httptest.NewRecorder()
	h.SaveConfig(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized payload, got %d", w.Code)
	}

	// 2. Empty ping targets
	emptyTargets := ConfigPayload{
		PingTargets:    []string{"", "  "},
		DNSTargets:     []string{"1.1.1.1"},
		PingIntervalS:  10,
		SpeedIntervalM: 5,
		PingCount:      5,
	}
	body, _ := json.Marshal(emptyTargets)
	req = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.SaveConfig(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty ping targets, got %d", w.Code)
	}

	// 3. Out-of-bounds ping count
	invalidPingCount := ConfigPayload{
		PingTargets:    []string{"1.1.1.1"},
		PingIntervalS:  10,
		SpeedIntervalM: 5,
		PingCount:      50, // Max is 20
	}
	body, _ = json.Marshal(invalidPingCount)
	req = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.SaveConfig(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for ping count > 20, got %d", w.Code)
	}
}

func TestServerTriggerUpdate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "update_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := monitor.New(monitor.DefaultConfig(), s, logger)
	h := NewHandler(context.Background(), s, mon, "v1.0.0")

	req := httptest.NewRequest(http.MethodPost, "/api/update", nil)
	w := httptest.NewRecorder()
	h.TriggerUpdate(w, req)

	// When up to date or on Windows, should return 200 or 400 without crashing
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code from TriggerUpdate: %d", w.Code)
	}
}

func TestServerTriggerSpeedTest(t *testing.T) {
	// Mock speed test HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bytes.Repeat([]byte("S"), 100*1024))
		} else {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockServer.Close()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "speed_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	cfg := monitor.Config{
		DownloadURL:    mockServer.URL + "/__down?bytes=102400",
		UploadURL:      mockServer.URL + "/__up",
		SpeedSizeBytes: 32 * 1024,
		SpeedStreams:   1,
		PingInterval:   1 * time.Second,
		SpeedInterval:  30 * time.Minute,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := monitor.New(cfg, s, logger)
	h := NewHandler(context.Background(), s, mon, "v1.0.0")

	// 1. Trigger speed test via API
	req := httptest.NewRequest(http.MethodPost, "/api/speedtest/run", nil)
	w := httptest.NewRecorder()
	h.TriggerSpeedTest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from TriggerSpeedTest, got %d (body: %s)", w.Code, w.Body.String())
	}

	var res struct {
		Status    string          `json:"status"`
		SpeedTest store.SpeedTest `json:"speed_test"`
	}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Status != "ok" || res.SpeedTest.DownloadMbps <= 0 {
		t.Fatalf("expected status 'ok' with positive download_mbps, got: %+v", res)
	}

	// 2. Query /api/data to verify the latest speed test is returned
	reqData := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	wData := httptest.NewRecorder()
	h.GetData(wData, reqData)

	if wData.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetData, got %d", wData.Code)
	}
	var dData DashboardData
	if err := json.NewDecoder(wData.Body).Decode(&dData); err != nil {
		t.Fatalf("failed to decode DashboardData: %v", err)
	}
	if dData.SpeedTest == nil || dData.SpeedTest.Time != res.SpeedTest.Time {
		t.Fatalf("expected speed test in DashboardData to match triggered test, got: %+v", dData.SpeedTest)
	}
}
