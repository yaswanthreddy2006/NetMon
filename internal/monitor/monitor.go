package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"netmon/internal/network"
	"netmon/internal/store"
)

type Config struct {
	PingTargets    []string
	DNSTargets     []string
	PingInterval   time.Duration
	SpeedInterval  time.Duration
	PingCount      int
	DownloadURL    string
	UploadURL      string
	SpeedSizeBytes int64
	SpeedStreams   int
}

func DefaultConfig() Config {
	return Config{
		PingTargets:    []string{"google.com", "cloudflare.com"},
		DNSTargets:     []string{"google.com", "cloudflare.com"},
		PingInterval:   60 * time.Second,
		SpeedInterval:  30 * time.Minute,
		PingCount:      5,
		DownloadURL:    "https://speed.cloudflare.com/__down?bytes=5000000",
		UploadURL:      "https://speed.cloudflare.com/__up",
		SpeedSizeBytes: DefaultUploadBytes,
		SpeedStreams:   DefaultSpeedStreams,
	}
}

// ValidateConfig sanitizes configuration and guarantees positive, safe intervals and non-empty targets.
func ValidateConfig(cfg Config) Config {
	def := DefaultConfig()

	var cleanPing []string
	for _, t := range cfg.PingTargets {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanPing = append(cleanPing, trimmed)
		}
	}
	if len(cleanPing) == 0 {
		cleanPing = def.PingTargets
	}
	cfg.PingTargets = cleanPing

	var cleanDNS []string
	for _, t := range cfg.DNSTargets {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanDNS = append(cleanDNS, trimmed)
		}
	}
	if len(cleanDNS) == 0 {
		cleanDNS = def.DNSTargets
	}
	cfg.DNSTargets = cleanDNS

	if cfg.PingInterval <= 0 {
		cfg.PingInterval = def.PingInterval
	} else if cfg.PingInterval > 24*time.Hour {
		cfg.PingInterval = 24 * time.Hour
	}

	if cfg.SpeedInterval <= 0 {
		cfg.SpeedInterval = def.SpeedInterval
	} else if cfg.SpeedInterval > 168*time.Hour {
		cfg.SpeedInterval = 168 * time.Hour
	}

	if cfg.PingCount <= 0 {
		cfg.PingCount = def.PingCount
	} else if cfg.PingCount > 20 {
		cfg.PingCount = 20
	}

	if strings.TrimSpace(cfg.DownloadURL) == "" {
		cfg.DownloadURL = def.DownloadURL
	}
	if strings.TrimSpace(cfg.UploadURL) == "" {
		cfg.UploadURL = def.UploadURL
	}
	if cfg.SpeedSizeBytes <= 0 {
		cfg.SpeedSizeBytes = def.SpeedSizeBytes
	}
	if cfg.SpeedStreams <= 0 {
		cfg.SpeedStreams = def.SpeedStreams
	}

	return cfg
}

type Monitor struct {
	cfgMu sync.RWMutex
	cfg   Config

	pingResetChan  chan time.Duration
	speedResetChan chan time.Duration

	store *store.Store
	log   *slog.Logger
	wg    sync.WaitGroup

	mu               sync.RWMutex
	lastDown         float64
	lastUp           float64
	currentNetworkID string

	trafficMu   sync.RWMutex
	liveTraffic network.LiveTraffic

	appTrafficMu sync.RWMutex
	appTraffic   network.ApplicationTraffic

	hotspotMu sync.RWMutex
	hotspot   network.HotspotState

	connMu    sync.RWMutex
	connInfo  network.ConnectionInfo
	localIP   string
	gatewayIP string

	isSpeedTesting atomic.Bool
}

func New(cfg Config, s *store.Store, log *slog.Logger) *Monitor {
	if log == nil {
		log = slog.Default()
	}
	validCfg := ValidateConfig(cfg)
	initConn, _ := network.GetConnectionInfo()
	initNet := network.Detect()
	return &Monitor{
		cfg:            validCfg,
		pingResetChan:  make(chan time.Duration, 10),
		speedResetChan: make(chan time.Duration, 10),
		store:          s,
		log:            log,
		connInfo:       initConn,
		localIP:        initNet.LocalIP,
		gatewayIP:      initNet.Gateway,
	}
}

// GetConnectionDetails returns the latest connection info, local host IP, and gateway IP.
func (m *Monitor) GetConnectionDetails() (network.ConnectionInfo, string, string) {
	m.connMu.RLock()
	defer m.connMu.RUnlock()
	return m.connInfo, m.localIP, m.gatewayIP
}

// GetConfig returns a snapshot of the current config.
func (m *Monitor) GetConfig() Config {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.cfg
}

// SetConfig replaces the live config and dynamically resets worker intervals immediately.
func (m *Monitor) SetConfig(cfg Config) {
	validCfg := ValidateConfig(cfg)

	m.cfgMu.Lock()
	oldPing := m.cfg.PingInterval
	oldSpeed := m.cfg.SpeedInterval
	m.cfg = validCfg
	m.cfgMu.Unlock()

	if validCfg.PingInterval != oldPing && validCfg.PingInterval > 0 {
		select {
		case m.pingResetChan <- validCfg.PingInterval:
		default:
		}
	}
	if validCfg.SpeedInterval != oldSpeed && validCfg.SpeedInterval > 0 {
		select {
		case m.speedResetChan <- validCfg.SpeedInterval:
		default:
		}
	}
}

// GetLiveTraffic returns the latest calculated real-time interface throughput.
func (m *Monitor) GetLiveTraffic() network.LiveTraffic {
	m.trafficMu.RLock()
	defer m.trafficMu.RUnlock()
	return m.liveTraffic
}

// GetApplicationTraffic returns the latest calculated real-time application bandwidth snapshot.
func (m *Monitor) GetApplicationTraffic() network.ApplicationTraffic {
	m.appTrafficMu.RLock()
	defer m.appTrafficMu.RUnlock()
	return m.appTraffic
}

// GetHotspotStatus returns the latest real-time mobile hotspot status and connected clients.
func (m *Monitor) GetHotspotStatus() network.HotspotState {
	m.hotspotMu.RLock()
	defer m.hotspotMu.RUnlock()
	return m.hotspot
}

// ConfigFromStore converts persisted settings into a full Config,
// filling URL fields from the compile-time defaults.
func ConfigFromStore(cs store.ConfigSettings) Config {
	def := DefaultConfig()
	cfg := Config{
		PingTargets:    cs.PingTargets,
		DNSTargets:     cs.DNSTargets,
		PingInterval:   time.Duration(cs.PingIntervalS) * time.Second,
		SpeedInterval:  time.Duration(cs.SpeedIntervalM) * time.Minute,
		PingCount:      cs.PingCount,
		DownloadURL:    def.DownloadURL,
		UploadURL:      def.UploadURL,
		SpeedSizeBytes: def.SpeedSizeBytes,
		SpeedStreams:   def.SpeedStreams,
	}
	return ValidateConfig(cfg)
}

func (m *Monitor) Start(ctx context.Context) {
	m.wg.Add(4)

	go func() {
		defer m.wg.Done()
		m.pingWorker(ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.speedWorker(ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.trafficWorker(ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.retentionWorker(ctx)
	}()
}

func (m *Monitor) Wait() {
	m.wg.Wait()
}

func (m *Monitor) trafficWorker(ctx context.Context) {
	prevStats, _ := network.ReadActiveInterfaceStats()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 1. Interface-level network traffic
			currStats, err := network.ReadActiveInterfaceStats()
			if err != nil {
				// Interface disappeared or disconnected
				prevStats = network.InterfaceStats{}
				m.trafficMu.Lock()
				m.liveTraffic = network.LiveTraffic{
					Interface:  "none",
					RxMbps:     0,
					TxMbps:     0,
					RxBytesSec: 0,
					TxBytesSec: 0,
					SampledAt:  time.Now().Format(time.RFC3339),
				}
				m.trafficMu.Unlock()
			} else {
				if prevStats.Name != "" && currStats.Name == prevStats.Name {
					rxMbps, txMbps, rxBps, txBps := network.CalculateThroughput(prevStats, currStats)
					m.trafficMu.Lock()
					m.liveTraffic = network.LiveTraffic{
						Interface:  currStats.Name,
						RxMbps:     rxMbps,
						TxMbps:     txMbps,
						RxBytesSec: rxBps,
						TxBytesSec: txBps,
						SampledAt:  currStats.Timestamp.Format(time.RFC3339),
					}
					m.trafficMu.Unlock()
				}
				prevStats = currStats
			}

			// 2. Application-level network traffic
			appTraffic := network.SampleActiveAppTraffic()
			m.appTrafficMu.Lock()
			m.appTraffic = appTraffic
			m.appTrafficMu.Unlock()

			// 3. Mobile Hotspot status & connected devices
			hotspotStatus := network.SampleActiveHotspotStatus()
			m.hotspotMu.Lock()
			m.hotspot = hotspotStatus
			m.hotspotMu.Unlock()
		}
	}
}

func (m *Monitor) pingWorker(ctx context.Context) {
	m.runPingCycle(ctx)

	cfg := m.GetConfig()
	ticker := time.NewTicker(cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("ping worker stopped")
			return
		case newDur := <-m.pingResetChan:
			ticker.Reset(newDur)
			m.log.Info("ping worker interval updated", "new_interval", newDur)
		case <-ticker.C:
			m.runPingCycle(ctx)
		}
	}
}

func (m *Monitor) speedWorker(ctx context.Context) {
	m.runSpeedCycle(ctx)

	cfg := m.GetConfig()
	ticker := time.NewTicker(cfg.SpeedInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("speed worker stopped")
			return
		case newDur := <-m.speedResetChan:
			ticker.Reset(newDur)
			m.log.Info("speed worker interval updated", "new_interval", newDur)
		case <-ticker.C:
			m.runSpeedCycle(ctx)
		}
	}
}

func (m *Monitor) retentionWorker(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once on startup
	if err := m.store.PruneOldRecords(30); err != nil {
		m.log.Error("prune old records failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.store.PruneOldRecords(30); err != nil {
				m.log.Error("prune old records failed", "error", err)
			}
		}
	}
}

func (m *Monitor) runPingCycle(ctx context.Context) {
	fmt.Println()
	m.log.Info("ping cycle: start")

	cfg := m.GetConfig() // snapshot config for this cycle

	// Detect current network; log an event if it changed.
	netInfo := network.Detect()
	m.mu.Lock()
	if netInfo.ID != m.currentNetworkID {
		if m.currentNetworkID != "" {
			m.log.Info("network changed", "from", m.currentNetworkID, "to", netInfo.ID)
		}
		m.currentNetworkID = netInfo.ID
		if err := m.store.LogNetworkEvent(store.NetworkEvent{
			Time:    time.Now().Format(time.RFC3339),
			Network: netInfo.ID,
			SSID:    netInfo.SSID,
			Gateway: netInfo.Gateway,
		}); err != nil {
			m.log.Error("log network event failed", "error", err)
		}
	}
	currentNetID := m.currentNetworkID
	m.mu.Unlock()

	allTargets := make([]string, 0, len(cfg.PingTargets)+1)
	if netInfo.Gateway != "" {
		allTargets = append(allTargets, netInfo.Gateway)
	}
	allTargets = append(allTargets, cfg.PingTargets...)

	results := make([]pingResult, len(allTargets))
	var wg sync.WaitGroup
	var connInfo network.ConnectionInfo

	wg.Add(1)
	go func() {
		defer wg.Done()
		connInfo, _ = network.GetConnectionInfo()
	}()

	for i, target := range allTargets {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			res := runDetailedPing(ctx, t, cfg.PingCount)
			results[idx] = res
		}(i, target)
	}

	dnsResults := make([]float64, len(cfg.DNSTargets))
	for i, host := range cfg.DNSTargets {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			dnsRes := network.MeasureDNSQuery(ctx, h, "system")
			if dnsRes.Status == network.QualitySuccess {
				dnsResults[idx] = dnsRes.DurationMs
			} else {
				m.log.Error("dns failed", "host", h, "status", dnsRes.Status, "error", dnsRes.Error)
			}
			m.store.UpsertDNSCheck(store.DNSCheck{
				Host:      h,
				TimeMs:    round1(dnsRes.DurationMs),
				Resolver:  "system",
				ServerIP:  dnsRes.ServerIP,
				Status:    string(dnsRes.Status),
				RCodeName: dnsRes.RCodeName,
				Error:     dnsRes.Error,
			})
		}(i, host)
	}

	wg.Wait()

	m.connMu.Lock()
	m.connInfo = connInfo
	m.localIP = netInfo.LocalIP
	m.gatewayIP = netInfo.Gateway
	m.connMu.Unlock()

	for i, r := range results {
		if r.Err != nil {
			m.log.Error("ping failed", "target", r.Host, "error", r.Err)
		}
		host := r.Host
		if i == 0 && netInfo.Gateway != "" {
			host = "gateway"
		}
		m.store.UpsertPingTarget(store.PingTarget{
			Host:        host,
			IP:          r.IP,
			Latency:     round1(r.AvgMs),
			MinLatency:  round1(r.MinMs),
			MaxLatency:  round1(r.MaxMs),
			AvgLatency:  round1(r.AvgMs),
			Jitter:      round1(r.JitterMs),
			Loss:        r.PacketLoss,
			ProbesSent:  r.ProbesSent,
			ProbesRecv:  r.ProbesRecv,
			ProbesLost:  r.ProbesLost,
			Method:      r.Method,
			Status:      r.Status,
		})
		m.log.Info("ping", "target", host, "latency_ms", r.AvgMs, "loss_%", r.PacketLoss, "method", r.Method)
	}

	start := 0
	if netInfo.Gateway != "" {
		start = 1
	}
	var totalLat, totalJitter, totalLoss float64
	var cnt int
	var validLatCnt int
	for _, r := range results[start:] {
		totalLoss += r.PacketLoss
		cnt++
		if r.Err == nil && r.PacketLoss < 100 && r.AvgMs > 0 {
			totalLat += r.AvgMs
			totalJitter += r.JitterMs
			validLatCnt++
		}
	}

	var dnsAvg float64
	var validDnsCnt int
	for _, d := range dnsResults {
		if d > 0 {
			dnsAvg += d
			validDnsCnt++
		}
	}
	if validDnsCnt > 0 {
		dnsAvg /= float64(validDnsCnt)
	}

	var avgLat, avgJitter float64
	if validLatCnt > 0 {
		avgLat = totalLat / float64(validLatCnt)
		avgJitter = totalJitter / float64(validLatCnt)
	}
	avgLoss := totalLoss / float64(max(cnt, 1))

	meas := store.Measurement{
		Time:         time.Now().Format(time.RFC3339),
		NetworkID:    currentNetID,
		Latency:      round1(avgLat),
		Jitter:       round1(avgJitter),
		PacketLoss:   round1(avgLoss),
		Download:     0,
		Upload:       0,
		DNS:          round1(dnsAvg),
		ConnType:     connInfo.Type,
		ConnRSSI:     connInfo.RSSI,
		ConnNoise:    connInfo.Noise,
		ConnSNR:      connInfo.SNR,
		ConnChannel:  connInfo.Channel,
		ConnBand:     connInfo.Band,
		ConnLinkRate: connInfo.LinkRate,
		ConnDuplex:   connInfo.Duplex,
	}

	if err := m.store.SaveMeasurement(meas); err != nil {
		m.log.Error("save measurement failed", "error", err)
	}
	m.log.Info("ping cycle: done", "latency", meas.Latency, "loss", meas.PacketLoss, "network", meas.NetworkID)
}

// RunSpeedTest executes an immediate manual speed test cycle, preventing concurrent executions.
func (m *Monitor) RunSpeedTest(ctx context.Context) (store.SpeedTest, error) {
	if !m.isSpeedTesting.CompareAndSwap(false, true) {
		return store.SpeedTest{}, fmt.Errorf("speed test already in progress")
	}
	defer m.isSpeedTesting.Store(false)

	return m.executeSpeedTest(ctx)
}

func (m *Monitor) runSpeedCycle(ctx context.Context) {
	if !m.isSpeedTesting.CompareAndSwap(false, true) {
		m.log.Info("speed test skipped: test already in progress")
		return
	}
	defer m.isSpeedTesting.Store(false)

	_, _ = m.executeSpeedTest(ctx)
}

func (m *Monitor) executeSpeedTest(ctx context.Context) (store.SpeedTest, error) {
	testTs := time.Now().Format(time.RFC3339)
	m.log.Info("speed test started", "timestamp", testTs)
	cfg := m.GetConfig()

	streams := cfg.SpeedStreams
	if streams < 1 {
		streams = DefaultSpeedStreams
	}
	upBytes := cfg.SpeedSizeBytes
	if upBytes < 1 {
		upBytes = DefaultUploadBytes
	}

	m.log.Info("download started", "url", cfg.DownloadURL, "streams", streams)
	down, bytesDown, durDown, errDown := measureDownload(ctx, cfg.DownloadURL, streams)
	if errDown != nil {
		m.log.Error("download failed", "error", errDown)
	} else {
		m.log.Info("download completed",
			"download_mbps", math.Round(down*100)/100,
			"download_bytes", bytesDown,
			"download_duration", math.Round(durDown*100)/100,
		)
	}

	m.log.Info("upload started", "url", cfg.UploadURL, "streams", streams, "target_bytes", upBytes)
	up, bytesUp, durUp, errUp := measureUpload(ctx, cfg.UploadURL, upBytes, streams)
	if errUp != nil {
		m.log.Error("upload failed", "error", errUp)
	} else {
		m.log.Info("upload completed",
			"upload_mbps", math.Round(up*100)/100,
			"upload_bytes", bytesUp,
			"upload_duration", math.Round(durUp*100)/100,
		)
	}

	status := "ok"
	var errMsg string
	if errDown != nil || errUp != nil {
		if errDown != nil && errUp != nil {
			status = "failed"
			errMsg = fmt.Sprintf("down: %v; up: %v", errDown, errUp)
		} else if errDown != nil {
			status = "partial"
			errMsg = fmt.Sprintf("down: %v", errDown)
		} else {
			status = "partial"
			errMsg = fmt.Sprintf("up: %v", errUp)
		}
	}

	st := store.SpeedTest{
		Time:            testTs,
		DownloadMbps:    math.Round(down*100) / 100,
		UploadMbps:      math.Round(up*100) / 100,
		DurationSeconds: math.Round((durDown+durUp)*100) / 100,
		BytesDown:       bytesDown,
		BytesUp:         bytesUp,
		Status:          status,
		Error:           errMsg,
	}

	if err := m.store.SaveSpeedTest(st); err != nil {
		m.log.Error("save speed test failed", "error", err)
	}

	m.mu.Lock()
	if down > 0 {
		m.lastDown = st.DownloadMbps
	}
	if up > 0 {
		m.lastUp = st.UploadMbps
	}
	m.mu.Unlock()

	m.log.Info("speed test finished",
		"timestamp", st.Time,
		"status", st.Status,
		"download_mbps", st.DownloadMbps,
		"upload_mbps", st.UploadMbps,
		"duration_s", st.DurationSeconds,
		"error", st.Error,
	)

	if status == "failed" {
		return st, fmt.Errorf("speed test failed: %s", errMsg)
	}
	return st, nil
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
