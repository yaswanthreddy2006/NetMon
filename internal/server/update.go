package server

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const maxBinarySize = 100 << 20 // 100 MB

func (h *Handler) TriggerUpdate(w http.ResponseWriter, r *http.Request) {
	vr := h.version.response()
	if !vr.UpdateAvailable {
		writeJSON(w, http.StatusOK, map[string]string{"status": "up_to_date"})
		return
	}

	if runtime.GOOS == "windows" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "self-update is not supported on Windows; download the latest release manually"})
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot determine executable path: " + err.Error()})
		return
	}

	tag := vr.Latest
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archiveName := fmt.Sprintf("netmon_%s_%s.tar.gz", goos, goarch)
	archiveURL := fmt.Sprintf("https://github.com/yaswanthreddy2006/NetMon/releases/download/%s/%s", tag, archiveName)
	checksumURL := fmt.Sprintf("https://github.com/yaswanthreddy2006/NetMon/releases/download/%s/checksums.txt", tag)

	// Respond before restarting so the client gets the response.
	writeJSON(w, http.StatusOK, map[string]string{"status": "updating", "version": tag})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := downloadAndReplace(archiveURL, checksumURL, archiveName, execPath); err != nil {
			return
		}
		restart(execPath)
	}()
}

func fetchExpectedChecksum(checksumURL, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NetMon/Update")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) == 2 && (fields[1] == filename || strings.TrimPrefix(fields[1], "*") == filename) {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found in checksums.txt", filename)
}

func downloadAndReplace(archiveURL, checksumURL, archiveName, execPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "NetMon/Update")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, ".netmon-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		tmp.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	tee := io.TeeReader(resp.Body, hasher)

	gz, err := gzip.NewReader(io.LimitReader(tee, maxBinarySize))
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) == "netmon" && hdr.Typeflag == tar.TypeReg {
			if _, err := io.Copy(tmp, io.LimitReader(tr, maxBinarySize)); err != nil {
				return fmt.Errorf("extract: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("netmon binary not found in release archive")
	}

	// Read rest of archive to finish computing sha256 of archive if needed
	_, _ = io.Copy(io.Discard, tee)

	// Verify SHA256 checksum if available
	if expectedHash, err := fetchExpectedChecksum(checksumURL, archiveName); err == nil && expectedHash != "" {
		computedHash := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(computedHash, expectedHash) {
			return fmt.Errorf("checksum mismatch (computed %s, expected %s)", computedHash, expectedHash)
		}
	}

	if err := tmp.Chmod(0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	tmp.Close()

	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("replace binary (check write permissions): %w", err)
	}
	success = true
	return nil
}
