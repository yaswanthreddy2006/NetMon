package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type versionChecker struct {
	current   string
	mu        sync.RWMutex
	latest    string
	lastFetch time.Time
}

func newVersionChecker(ctx context.Context, current string) *versionChecker {
	vc := &versionChecker{current: current}
	if ctx == nil {
		ctx = context.Background()
	}
	go vc.fetch(ctx)
	go vc.periodicCheck(ctx)
	return vc
}

func (vc *versionChecker) periodicCheck(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			vc.fetch(ctx)
		}
	}
}

func (vc *versionChecker) fetch(ctx context.Context) {
	vc.mu.Lock()
	if time.Since(vc.lastFetch) < 30*time.Second {
		vc.mu.Unlock()
		return
	}
	vc.lastFetch = time.Now()
	vc.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://api.github.com/repos/yaswanthreddy2006/NetMon/releases/latest", nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "NetMon/"+vc.current)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return
	}

	vc.mu.Lock()
	if payload.TagName != "" {
		vc.latest = payload.TagName
	}
	vc.mu.Unlock()
}

type VersionResponse struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
}

func (vc *versionChecker) response() VersionResponse {
	vc.mu.RLock()
	latest := vc.latest
	vc.mu.RUnlock()
	return VersionResponse{
		Current:         vc.current,
		Latest:          latest,
		UpdateAvailable: latest != "" && latest != vc.current,
	}
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("check") == "true" || r.URL.Query().Get("check") == "1" {
		h.version.fetch(r.Context())
	}
	writeJSON(w, http.StatusOK, h.version.response())
}
