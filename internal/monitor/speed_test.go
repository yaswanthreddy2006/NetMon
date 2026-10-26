package monitor

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"netmon/internal/store"
)

func TestSpeedUnitConversions(t *testing.T) {
	// 10,000,000 bytes in 1.0 second:
	// Bits: 80,000,000 bits / 1s = 80.0 Mbps
	// MB/s: 10.0 MB/s
	// MiB/s: 10,000,000 / 1048576 = 9.536743 MiB/s
	bytesCount := int64(10000000)
	durSec := 1.0

	mbps := BytesToMbps(bytesCount, durSec)
	if math.Abs(mbps-80.0) > 1e-4 {
		t.Fatalf("expected 80.0 Mbps, got %f", mbps)
	}

	mbs := BytesToMBs(bytesCount, durSec)
	if math.Abs(mbs-10.0) > 1e-4 {
		t.Fatalf("expected 10.0 MB/s, got %f", mbs)
	}

	mibs := BytesToMiBs(bytesCount, durSec)
	expectedMibs := float64(bytesCount) / (1024 * 1024)
	if math.Abs(mibs-expectedMibs) > 1e-4 {
		t.Fatalf("expected %f MiB/s, got %f", expectedMibs, mibs)
	}

	// Zero duration safety
	if BytesToMbps(bytesCount, 0) != 0 {
		t.Fatalf("expected 0 for 0 duration")
	}
}

func TestMeasureDownloadMultiStream(t *testing.T) {
	streamPayload := 500 * 1024 // 500 KB per stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		data := bytes.Repeat([]byte("X"), 32*1024)
		for i := 0; i < streamPayload/(32*1024); i++ {
			w.Write(data)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mbps, totalBytes, dur, err := measureDownload(ctx, server.URL, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalBytes < int64(streamPayload) {
		t.Fatalf("expected at least %d bytes, got %d", streamPayload, totalBytes)
	}
	if mbps <= 0 {
		t.Fatalf("expected positive mbps, got %f", mbps)
	}
	if dur <= 0 {
		t.Fatalf("expected positive duration, got %f", dur)
	}
}

func TestMeasureDownloadHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, err := measureDownload(ctx, server.URL, 2)
	if err == nil {
		t.Fatalf("expected error on 500 status code, got nil")
	}
}

func TestMeasureUploadMultiStream(t *testing.T) {
	var totalReceived int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		totalReceived += n
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testBytesPerStream := int64(256 * 1024)
	mbps, totalBytes, dur, err := measureUpload(ctx, server.URL, testBytesPerStream, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalBytes != testBytesPerStream*2 {
		t.Fatalf("expected %d bytes sent, got %d", testBytesPerStream*2, totalBytes)
	}
	if mbps <= 0 {
		t.Fatalf("expected positive mbps, got %f", mbps)
	}
	if dur <= 0 {
		t.Fatalf("expected positive duration, got %f", dur)
	}
}

func TestPatternReader(t *testing.T) {
	pr := newPatternReader([]byte{1, 2, 3}, 7)
	buf := make([]byte, 4)

	n, err := pr.Read(buf)
	if err != nil || n != 4 {
		t.Fatalf("expected 4 bytes, got %d (err: %v)", n, err)
	}
	expected := []byte{1, 2, 3, 1}
	for i := range expected {
		if buf[i] != expected[i] {
			t.Fatalf("byte %d mismatch: got %d, expected %d", i, buf[i], expected[i])
		}
	}

	n, err = pr.Read(buf)
	if err != nil || n != 3 {
		t.Fatalf("expected 3 bytes remaining, got %d (err: %v)", n, err)
	}

	n, err = pr.Read(buf)
	if err != io.EOF || n != 0 {
		t.Fatalf("expected EOF on finished reader, got n=%d, err=%v", n, err)
	}
}

func TestMeasureSpeedCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, _, err := measureDownload(ctx, server.URL, 1)
	if err == nil {
		t.Fatalf("expected timeout/cancellation error, got nil")
	}

	_, _, _, err = measureUpload(ctx, server.URL, 64*1024, 1)
	if err == nil {
		t.Fatalf("expected timeout/cancellation error on upload, got nil")
	}
}

func TestMeasureDownloadPayloadTooSmall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("too small")) // Only 9 bytes
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, _, err := measureDownload(ctx, server.URL, 1)
	if err == nil {
		t.Fatalf("expected error for payload smaller than MinValidPayloadBytes, got nil")
	}
}

func TestMeasureDownloadHTTPStatuses(t *testing.T) {
	statuses := []int{
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusTooManyRequests,
	}

	for _, code := range statuses {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error", code)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, _, _, err := measureDownload(ctx, server.URL, 1)
			if err == nil {
				t.Fatalf("expected error for HTTP status %d, got nil", code)
			}
		})
	}
}

func TestMeasureUploadServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, _, err := measureUpload(ctx, server.URL, 64*1024, 1)
	if err == nil {
		t.Fatalf("expected error for HTTP 500 upload, got nil")
	}
}

func TestPrepareDownloadURL(t *testing.T) {
	u1 := prepareDownloadURL("https://speed.cloudflare.com/__down?bytes=1000", 5000000)
	if u1 != "https://speed.cloudflare.com/__down?bytes=5000000" {
		t.Fatalf("expected updated bytes query param, got %s", u1)
	}

	u2 := prepareDownloadURL("https://speed.cloudflare.com/__down", 5000000)
	if u2 != "https://speed.cloudflare.com/__down" {
		t.Fatalf("expected untouched URL when no bytes query param exists, got %s", u2)
	}

	u3 := prepareDownloadURL("not a valid url ://", 5000000)
	if u3 != "not a valid url ://" {
		t.Fatalf("expected original string on parse error, got %s", u3)
	}
}

func TestMonitorRunSpeedTestAndPreventStaleData(t *testing.T) {
	// Setup mock HTTP server for speed test
	downloadPayload := 200 * 1024 // 200 KB
	var uploadCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bytes.Repeat([]byte("Z"), downloadPayload))
		} else if r.Method == http.MethodPost {
			n, _ := io.Copy(io.Discard, r.Body)
			uploadCount += n
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// In-memory store
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	defer st.Close()

	cfg := Config{
		DownloadURL:    server.URL + "/__down?bytes=204800",
		UploadURL:      server.URL + "/__up",
		SpeedSizeBytes: 64 * 1024,
		SpeedStreams:   1,
		PingInterval:   1 * time.Second,
		SpeedInterval:  30 * time.Minute,
	}

	mon := New(cfg, st, nil)

	// 1. Run first successful speed test
	res1, err := mon.RunSpeedTest(context.Background())
	if err != nil {
		t.Fatalf("expected speed test success, got: %v", err)
	}
	if res1.Status != "ok" || res1.DownloadMbps <= 0 || res1.UploadMbps <= 0 {
		t.Fatalf("expected successful speed test with positive mbps, got: %+v", res1)
	}

	// Verify database record
	dbLatest, err := st.GetLatestSpeedTest()
	if err != nil || dbLatest.Status != "ok" {
		t.Fatalf("expected latest test in db to be ok, got: %+v, err: %v", dbLatest, err)
	}
	if dbLatest.Time != res1.Time {
		t.Fatalf("db timestamp mismatch: %s vs %s", dbLatest.Time, res1.Time)
	}

	// 2. Test concurrent speed test prevention (atomic lock)
	mon.isSpeedTesting.Store(true)
	_, err = mon.RunSpeedTest(context.Background())
	if err == nil || err.Error() != "speed test already in progress" {
		t.Fatalf("expected 'speed test already in progress' error, got: %v", err)
	}
	mon.isSpeedTesting.Store(false)

	// 3. Test failed speed test does not produce fake speed and records failed status
	brokenCfg := cfg
	brokenCfg.DownloadURL = "http://127.0.0.1:9999/unreachable"
	brokenCfg.UploadURL = "http://127.0.0.1:9999/unreachable"
	mon.SetConfig(brokenCfg)

	res2, err := mon.RunSpeedTest(context.Background())
	if err == nil {
		t.Fatalf("expected error on failed download/upload URL, got nil")
	}
	if res2.Status != "failed" {
		t.Fatalf("expected status 'failed', got %s", res2.Status)
	}

	// Verify db now returns the failed test as the latest test (so UI knows it failed rather than showing stale)
	dbLatestFailed, err := st.GetLatestSpeedTest()
	if err != nil {
		t.Fatalf("failed to get latest test: %v", err)
	}
	if dbLatestFailed.Status != "failed" {
		t.Fatalf("expected latest test to reflect failed status, got %s", dbLatestFailed.Status)
	}
	if dbLatestFailed.Time != res2.Time {
		t.Fatalf("expected failed test timestamp %s, got %s", res2.Time, dbLatestFailed.Time)
	}
}
