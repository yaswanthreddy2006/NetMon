package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"netmon/internal/monitor"
	"netmon/internal/network"
	"netmon/internal/store"
)

type Handler struct {
	store   *store.Store
	mon     *monitor.Monitor
	version *versionChecker
}

func NewHandler(ctx context.Context, s *store.Store, mon *monitor.Monitor, version string) *Handler {
	return &Handler{store: s, mon: mon, version: newVersionChecker(ctx, version)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type DashboardData struct {
	ServerTime         string                     `json:"server_time"`
	Summary            store.Summary              `json:"summary"`
	Targets            []store.PingTarget         `json:"targets"`
	DNS                []store.DNSCheck           `json:"dns"`
	History            []store.Measurement        `json:"history"`
	LatestMeasurement  *store.Measurement         `json:"latest_measurement,omitempty"`
	SpeedTests         []store.SpeedTest          `json:"speed_tests"`
	SpeedTest          *store.SpeedTest           `json:"speed_test,omitempty"`
	LiveTraffic        network.LiveTraffic        `json:"live_traffic"`
	ApplicationTraffic network.ApplicationTraffic `json:"application_traffic"`
	Hotspot            network.HotspotState       `json:"hotspot"`
	Diagnosis          monitor.DiagnosisResult    `json:"diagnosis"`
	NetworkID          string                     `json:"network_id"`
	ConnectionInfo     *network.ConnectionInfo    `json:"connection_info,omitempty"`
	LocalIP            string                     `json:"local_ip,omitempty"`
	GatewayIP          string                     `json:"gateway_ip,omitempty"`
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	minutes := 60
	if m := r.URL.Query().Get("minutes"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 && n <= 43200 {
			minutes = n
		}
	}

	summary, err := h.store.GetSummary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	targets, err := h.store.GetPingTargets()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	dns, err := h.store.GetDNSChecks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	history, err := h.store.GetHistoryWindow(minutes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	speedTests, err := h.store.GetSpeedTestsWindow(minutes)
	if err != nil {
		speedTests = []store.SpeedTest{}
	}

	latestSt, errSt := h.store.GetLatestSpeedTest()
	var stPtr *store.SpeedTest
	if errSt == nil && latestSt.Time != "" {
		stPtr = &latestSt
	}

	latestMeas, errMeas := h.store.GetLatestMeasurement()
	var measPtr *store.Measurement
	if errMeas == nil && latestMeas.Timestamp != "" {
		measPtr = &latestMeas
	}

	liveTraffic := h.mon.GetLiveTraffic()
	appTraffic := h.mon.GetApplicationTraffic()
	hotspot := h.mon.GetHotspotStatus()

	recentEvents, _ := h.store.GetRecentNetworkEvents(5)

	diagnosis := monitor.Diagnose(monitor.DiagnosticInput{
		Targets:      targets,
		DNSChecks:    dns,
		Summary:      summary,
		LiveTraffic:  liveTraffic,
		RecentEvents: recentEvents,
	})

	connDetails, localIP, gatewayIP := h.mon.GetConnectionDetails()
	var connPtr *network.ConnectionInfo
	if connDetails.Status != "" && connDetails.Status != "unavailable" {
		connPtr = &connDetails
	}

	netID := h.store.GetCurrentNetworkID()
	if netID == "" && connDetails.SSID != "" {
		netID = connDetails.SSID
	}

	writeJSON(w, http.StatusOK, DashboardData{
		ServerTime:         time.Now().Format(time.RFC3339),
		Summary:            summary,
		Targets:            targets,
		DNS:                dns,
		History:            history,
		LatestMeasurement:  measPtr,
		SpeedTests:         speedTests,
		SpeedTest:          stPtr,
		LiveTraffic:        liveTraffic,
		ApplicationTraffic: appTraffic,
		Hotspot:            hotspot,
		Diagnosis:          diagnosis,
		NetworkID:          netID,
		ConnectionInfo:     connPtr,
		LocalIP:            localIP,
		GatewayIP:          gatewayIP,
	})
}

// ConfigPayload is the API shape for GET /api/config and POST /api/config.
type ConfigPayload struct {
	PingTargets    []string `json:"ping_targets"`
	DNSTargets     []string `json:"dns_targets"`
	PingIntervalS  int      `json:"ping_interval_s"`
	SpeedIntervalM int      `json:"speed_interval_m"`
	PingCount      int      `json:"ping_count"`
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.mon.GetConfig()
	writeJSON(w, http.StatusOK, ConfigPayload{
		PingTargets:    cfg.PingTargets,
		DNSTargets:     cfg.DNSTargets,
		PingIntervalS:  int(cfg.PingInterval.Seconds()),
		SpeedIntervalM: int(cfg.SpeedInterval.Minutes()),
		PingCount:      cfg.PingCount,
	})
}

func (h *Handler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var p ConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Clean and validate targets
	var cleanPing []string
	for _, t := range p.PingTargets {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanPing = append(cleanPing, trimmed)
		}
	}
	if len(cleanPing) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ping_targets cannot be empty"})
		return
	}

	var cleanDNS []string
	for _, t := range p.DNSTargets {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanDNS = append(cleanDNS, trimmed)
		}
	}
	if len(cleanDNS) == 0 {
		cleanDNS = []string{"1.1.1.1", "8.8.8.8"}
	}

	if p.PingIntervalS < 5 || p.PingIntervalS > 86400 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ping_interval_s must be between 5 and 86400 seconds"})
		return
	}
	if p.SpeedIntervalM < 5 || p.SpeedIntervalM > 10080 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "speed_interval_m must be between 5 and 10080 minutes"})
		return
	}
	if p.PingCount < 1 || p.PingCount > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ping_count must be between 1 and 20"})
		return
	}

	p.PingTargets = cleanPing
	p.DNSTargets = cleanDNS

	if err := h.store.SaveConfig(store.ConfigSettings{
		PingTargets:    p.PingTargets,
		DNSTargets:     p.DNSTargets,
		PingIntervalS:  p.PingIntervalS,
		SpeedIntervalM: p.SpeedIntervalM,
		PingCount:      p.PingCount,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	def := monitor.DefaultConfig()
	h.mon.SetConfig(monitor.Config{
		PingTargets:   p.PingTargets,
		DNSTargets:    p.DNSTargets,
		PingInterval:  time.Duration(p.PingIntervalS) * time.Second,
		SpeedInterval: time.Duration(p.SpeedIntervalM) * time.Minute,
		PingCount:     p.PingCount,
		DownloadURL:   def.DownloadURL,
		UploadURL:     def.UploadURL,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *Handler) TriggerSpeedTest(w http.ResponseWriter, r *http.Request) {
	st, err := h.mon.RunSpeedTest(r.Context())
	if err != nil {
		if err.Error() == "speed test already in progress" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":      err.Error(),
			"speed_test": st,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"speed_test": st,
	})
}
