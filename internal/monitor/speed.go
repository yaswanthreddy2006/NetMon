package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	DefaultSpeedTimeout  = 15 * time.Second
	MinValidPayloadBytes = 5000            // Minimum bytes to consider a valid speed test payload
	DefaultDownloadBytes = 5 * 1000 * 1000 // 5 MB payload per stream
	DefaultUploadBytes   = 2 * 1024 * 1024 // 2 MB payload per stream
	DefaultSpeedStreams  = 2               // 2 concurrent streams to saturate connection
)

// SpeedResult holds detailed metrics for a speed test execution.
type SpeedResult struct {
	DownloadMbps    float64 `json:"download_mbps"`
	UploadMbps      float64 `json:"upload_mbps"`
	DurationSeconds float64 `json:"duration_seconds"`
	BytesDownloaded int64   `json:"bytes_down"`
	BytesUploaded   int64   `json:"bytes_up"`
	Status          string  `json:"status"`
	Error           string  `json:"error"`
}

// BytesToMbps converts transferred bytes and transfer duration into Megabits per second (SI decimal, 10^6 bps).
func BytesToMbps(byteCount int64, durationSeconds float64) float64 {
	if durationSeconds <= 0.0001 || byteCount <= 0 {
		return 0
	}
	bits := float64(byteCount) * 8.0
	return (bits / durationSeconds) / 1e6
}

// BytesToMBs converts transferred bytes and transfer duration into Megabytes per second (decimal, 10^6 B/s).
func BytesToMBs(byteCount int64, durationSeconds float64) float64 {
	if durationSeconds <= 0.0001 || byteCount <= 0 {
		return 0
	}
	return (float64(byteCount) / durationSeconds) / 1e6
}

// BytesToMiBs converts transferred bytes and transfer duration into Mebibytes per second (binary, 2^20 B/s).
func BytesToMiBs(byteCount int64, durationSeconds float64) float64 {
	if durationSeconds <= 0.0001 || byteCount <= 0 {
		return 0
	}
	return (float64(byteCount) / durationSeconds) / (1024.0 * 1024.0)
}

// newSpeedHTTPClient creates an HTTP client optimized for throughput testing.
func newSpeedHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    true, // Avoid gzip CPU decompression skewing transfer measurements
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// prepareDownloadURL ensures the download URL requests an adequate payload size.
func prepareDownloadURL(rawURL string, targetBytes int64) string {
	if targetBytes <= 0 {
		targetBytes = DefaultDownloadBytes
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if q.Has("bytes") {
		currentBytes, _ := strconv.ParseInt(q.Get("bytes"), 10, 64)
		if currentBytes < targetBytes {
			q.Set("bytes", strconv.FormatInt(targetBytes, 10))
			u.RawQuery = q.Encode()
		}
	}
	return u.String()
}

// measureDownloadStream executes a single download stream, measuring transfer time from first byte.
func measureDownloadStream(ctx context.Context, client *http.Client, targetURL string) (bytesRead int64, durSec float64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NetMon/1.0")
	req.Header.Set("Accept-Encoding", "identity")

	var transferStart time.Time
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			transferStart = time.Now()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("download returned non-2xx status: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	if transferStart.IsZero() {
		transferStart = time.Now()
	}

	buf := make([]byte, 64*1024)
	for {
		n, rErr := resp.Body.Read(buf)
		bytesRead += int64(n)
		if rErr == io.EOF {
			break
		}
		if rErr != nil {
			dur := time.Since(transferStart).Seconds()
			return bytesRead, dur, fmt.Errorf("download stream error: %w", rErr)
		}
	}

	durSec = time.Since(transferStart).Seconds()
	if durSec <= 0.001 {
		durSec = 0.001
	}

	if bytesRead < MinValidPayloadBytes {
		return bytesRead, durSec, fmt.Errorf("download payload too small (%d bytes), expected real data", bytesRead)
	}

	return bytesRead, durSec, nil
}

// measureDownload performs multi-stream download throughput measurement.
func measureDownload(ctx context.Context, targetURL string, streams int) (mbps float64, totalBytes int64, durSec float64, err error) {
	if streams < 1 {
		streams = DefaultSpeedStreams
	}
	targetURL = prepareDownloadURL(targetURL, DefaultDownloadBytes)

	client := newSpeedHTTPClient(DefaultSpeedTimeout)
	defer client.CloseIdleConnections()
	overallStart := time.Now()

	type streamResult struct {
		bytes int64
		dur   float64
		err   error
	}

	results := make([]streamResult, streams)
	var wg sync.WaitGroup

	for i := 0; i < streams; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b, d, e := measureDownloadStream(ctx, client, targetURL)
			results[idx] = streamResult{bytes: b, dur: d, err: e}
		}(i)
	}

	wg.Wait()
	durSec = time.Since(overallStart).Seconds()

	var successfulStreams int
	for _, r := range results {
		if r.err == nil && r.bytes >= MinValidPayloadBytes {
			totalBytes += r.bytes
			successfulStreams++
		}
	}

	if successfulStreams == 0 {
		var firstErr error
		for _, r := range results {
			if r.err != nil {
				firstErr = r.err
				break
			}
		}
		return 0, totalBytes, durSec, firstErr
	}

	if durSec <= 0.001 {
		durSec = 0.001
	}

	mbps = BytesToMbps(totalBytes, durSec)
	return mbps, totalBytes, durSec, nil
}

type patternReader struct {
	pattern []byte
	pos     int
	total   int64
	read    int64
}

func newPatternReader(pattern []byte, total int64) *patternReader {
	if len(pattern) == 0 {
		pattern = []byte{0xAA}
	}
	return &patternReader{
		pattern: pattern,
		total:   total,
	}
}

func (r *patternReader) Read(p []byte) (n int, err error) {
	if r.read >= r.total {
		return 0, io.EOF
	}
	remain := r.total - r.read
	toRead := int64(len(p))
	if toRead > remain {
		toRead = remain
	}
	plen := len(r.pattern)
	for i := int64(0); i < toRead; i++ {
		p[i] = r.pattern[r.pos]
		r.pos = (r.pos + 1) % plen
	}
	r.read += toRead
	return int(toRead), nil
}

// measureUploadStream executes a streaming upload chunk without allocating full payload in memory or leaking goroutines.
func measureUploadStream(ctx context.Context, client *http.Client, targetURL string, uploadBytes int64) (bytesSent int64, durSec float64, err error) {
	if uploadBytes <= 0 {
		uploadBytes = DefaultUploadBytes
	}

	reader := newPatternReader([]byte{0xAA, 0x55, 0xCC, 0x33}, uploadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, reader)
	if err != nil {
		return 0, 0, fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NetMon/1.0")
	req.ContentLength = uploadBytes

	start := time.Now()
	resp, err := client.Do(req)
	durSec = time.Since(start).Seconds()

	if err != nil {
		return 0, durSec, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, durSec, fmt.Errorf("upload returned non-2xx status: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	if durSec <= 0.001 {
		durSec = 0.001
	}

	return uploadBytes, durSec, nil
}

// measureUpload performs multi-stream upload throughput measurement.
func measureUpload(ctx context.Context, targetURL string, sizeBytes int64, streams int) (mbps float64, totalBytes int64, durSec float64, err error) {
	if streams < 1 {
		streams = DefaultSpeedStreams
	}
	if sizeBytes <= 0 {
		sizeBytes = DefaultUploadBytes
	}

	client := newSpeedHTTPClient(DefaultSpeedTimeout)
	defer client.CloseIdleConnections()
	overallStart := time.Now()

	type streamResult struct {
		bytes int64
		dur   float64
		err   error
	}

	results := make([]streamResult, streams)
	var wg sync.WaitGroup

	for i := 0; i < streams; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b, d, e := measureUploadStream(ctx, client, targetURL, sizeBytes)
			results[idx] = streamResult{bytes: b, dur: d, err: e}
		}(i)
	}

	wg.Wait()
	durSec = time.Since(overallStart).Seconds()

	var successfulStreams int
	for _, r := range results {
		if r.err == nil && r.bytes > 0 {
			totalBytes += r.bytes
			successfulStreams++
		}
	}

	if successfulStreams == 0 {
		for _, r := range results {
			if r.err != nil {
				return 0, totalBytes, durSec, r.err
			}
		}
		return 0, totalBytes, durSec, fmt.Errorf("all upload streams failed")
	}

	if durSec <= 0.001 {
		durSec = 0.001
	}

	mbps = BytesToMbps(totalBytes, durSec)
	return mbps, totalBytes, durSec, nil
}
