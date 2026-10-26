package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed queries.sql
var queriesRaw string

// q holds all named queries parsed from queries.sql.
var q = parseQueries(queriesRaw)

// parseQueries splits queries.sql into a map keyed by "-- name: <key>" markers.
func parseQueries(data string) map[string]string {
	result := make(map[string]string)
	var name string
	var lines []string
	for _, line := range strings.Split(data, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "-- name:") {
			if name != "" {
				result[name] = strings.TrimSpace(strings.Join(lines, "\n"))
			}
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- name:"))
			lines = nil
		} else if name != "" {
			lines = append(lines, line)
		}
	}
	if name != "" {
		result[name] = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return result
}

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000&_sync=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// SQLite requires single-writer serialization to completely avoid locking issues.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return err
	}
	// Safe incremental column additions for pre-existing databases (no-op if already present)
	_, _ = s.db.Exec(`ALTER TABLE measurements ADD COLUMN network_id TEXT NOT NULL DEFAULT ''`)

	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN min_latency REAL NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN max_latency REAL NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN avg_latency REAL NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN jitter REAL NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN probes_sent INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN probes_recv INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN probes_lost INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE ping_targets ADD COLUMN method TEXT NOT NULL DEFAULT 'icmp'`)

	_, _ = s.db.Exec(`ALTER TABLE dns_checks ADD COLUMN server_ip TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE dns_checks ADD COLUMN status TEXT NOT NULL DEFAULT 'SUCCESS'`)
	_, _ = s.db.Exec(`ALTER TABLE dns_checks ADD COLUMN rcode_name TEXT NOT NULL DEFAULT 'NOERROR'`)
	_, _ = s.db.Exec(`ALTER TABLE dns_checks ADD COLUMN error TEXT NOT NULL DEFAULT ''`)

	return nil
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// --- Measurement ---

type Measurement struct {
	ID           int64   `json:"id"`
	Time         string  `json:"time"`
	Timestamp    string  `json:"timestamp"`
	NetworkID    string  `json:"network_id"`
	Latency      float64 `json:"latency"`
	Jitter       float64 `json:"jitter"`
	PacketLoss   float64 `json:"loss"`
	Download     float64 `json:"download"`
	Upload       float64 `json:"upload"`
	DNS          float64 `json:"dns"`
	ConnType     string  `json:"conn_type"`
	ConnRSSI     int     `json:"conn_rssi"`
	ConnNoise    int     `json:"conn_noise"`
	ConnSNR      int     `json:"conn_snr"`
	ConnChannel  int     `json:"conn_channel"`
	ConnBand     string  `json:"conn_band"`
	ConnLinkRate int     `json:"conn_link_rate"`
	ConnDuplex   string  `json:"conn_duplex"`
}

func (s *Store) SaveMeasurement(m Measurement) error {
	_, err := s.db.Exec(q["insert_measurement"],
		m.Time, m.NetworkID, m.Latency, m.Jitter, m.PacketLoss, m.Download, m.Upload, m.DNS,
		m.ConnType, m.ConnRSSI, m.ConnNoise, m.ConnSNR, m.ConnChannel, m.ConnBand, m.ConnLinkRate, m.ConnDuplex,
	)
	return err
}

// --- SpeedTest ---

type SpeedTest struct {
	ID              int64   `json:"id"`
	Time            string  `json:"time"`
	DownloadMbps    float64 `json:"download_mbps"`
	UploadMbps      float64 `json:"upload_mbps"`
	DurationSeconds float64 `json:"duration_seconds"`
	BytesDown       int64   `json:"bytes_down"`
	BytesUp         int64   `json:"bytes_up"`
	Status          string  `json:"status"`
	Error           string  `json:"error"`
}

func (s *Store) SaveSpeedTest(st SpeedTest) error {
	_, err := s.db.Exec(q["insert_speed_test"],
		st.Time, st.DownloadMbps, st.UploadMbps, st.DurationSeconds, st.BytesDown, st.BytesUp, st.Status, st.Error,
	)
	return err
}

func (s *Store) GetSpeedTestsWindow(minutes int) ([]SpeedTest, error) {
	if minutes <= 0 {
		minutes = 60
	}
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute).Format(time.RFC3339)
	rows, err := s.db.Query(q["get_speed_tests_window"], cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SpeedTest
	for rows.Next() {
		var st SpeedTest
		if err := rows.Scan(
			&st.ID, &st.Time, &st.DownloadMbps, &st.UploadMbps, &st.DurationSeconds, &st.BytesDown, &st.BytesUp, &st.Status, &st.Error,
		); err != nil {
			return nil, err
		}
		results = append(results, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []SpeedTest{}
	}
	return results, nil
}

func (s *Store) GetLatestSpeedTest() (SpeedTest, error) {
	var st SpeedTest
	err := s.db.QueryRow(q["get_latest_speed_test"]).Scan(
		&st.ID, &st.Time, &st.DownloadMbps, &st.UploadMbps, &st.DurationSeconds, &st.BytesDown, &st.BytesUp, &st.Status, &st.Error,
	)
	if err == sql.ErrNoRows {
		return SpeedTest{}, nil
	}
	return st, err
}

func (s *Store) GetLatestMeasurement() (Measurement, error) {
	var m Measurement
	var ts string
	var lat, jit, loss, dl, ul, dns sql.NullFloat64
	err := s.db.QueryRow(q["get_latest_measurement"]).Scan(
		&m.ID, &ts, &m.NetworkID, &lat, &jit, &loss, &dl, &ul, &dns,
		&m.ConnType, &m.ConnRSSI, &m.ConnNoise, &m.ConnSNR, &m.ConnChannel, &m.ConnBand, &m.ConnLinkRate, &m.ConnDuplex,
	)
	if err == sql.ErrNoRows {
		return Measurement{}, nil
	}
	if err != nil {
		return Measurement{}, err
	}
	if lat.Valid {
		m.Latency = lat.Float64
	}
	if jit.Valid {
		m.Jitter = jit.Float64
	}
	if loss.Valid {
		m.PacketLoss = loss.Float64
	}
	if dl.Valid {
		m.Download = dl.Float64
	}
	if ul.Valid {
		m.Upload = ul.Float64
	}
	if dns.Valid {
		m.DNS = dns.Float64
	}
	m.Timestamp = ts
	m.Time = ts
	return m, nil
}

func (s *Store) GetHistory(limit int) ([]Measurement, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(q["get_history"], limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Measurement
	for rows.Next() {
		var m Measurement
		var ts string
		var lat, jit, loss, dl, ul, dns sql.NullFloat64
		if err := rows.Scan(
			&m.ID, &ts, &m.NetworkID, &lat, &jit, &loss, &dl, &ul, &dns,
			&m.ConnType, &m.ConnRSSI, &m.ConnNoise, &m.ConnSNR, &m.ConnChannel, &m.ConnBand, &m.ConnLinkRate, &m.ConnDuplex,
		); err != nil {
			return nil, err
		}
		if lat.Valid {
			m.Latency = lat.Float64
		}
		if jit.Valid {
			m.Jitter = jit.Float64
		}
		if loss.Valid {
			m.PacketLoss = loss.Float64
		}
		if dl.Valid {
			m.Download = dl.Float64
		}
		if ul.Valid {
			m.Upload = ul.Float64
		}
		if dns.Valid {
			m.DNS = dns.Float64
		}

		m.Timestamp = ts
		t, _ := time.Parse(time.RFC3339, ts)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02T15:04:05Z", ts)
		}
		m.Time = t.Format("15:04:05")
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	if results == nil {
		results = []Measurement{}
	}
	return results, nil
}

func (s *Store) GetHistoryWindow(minutes int) ([]Measurement, error) {
	if minutes <= 0 {
		minutes = 60
	}
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute).Format(time.RFC3339)

	var queryName string
	var timeFmt string
	switch {
	case minutes > 10080: // > 7 days -> 1 hour downsampling
		queryName = "get_history_downsampled_1h"
		timeFmt = "Jan 02 15:04"
	case minutes > 1440: // > 24 hours -> 5 min downsampling
		queryName = "get_history_downsampled_5m"
		timeFmt = "Jan 02 15:04"
	case minutes > 60:
		queryName = "get_history_window"
		timeFmt = "15:04"
	default:
		queryName = "get_history_window"
		timeFmt = "15:04:05"
	}

	rows, err := s.db.Query(q[queryName], cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Measurement
	for rows.Next() {
		var m Measurement
		var ts string
		var lat, jit, loss, dl, ul, dns sql.NullFloat64
		if err := rows.Scan(
			&m.ID, &ts, &m.NetworkID, &lat, &jit, &loss, &dl, &ul, &dns,
			&m.ConnType, &m.ConnRSSI, &m.ConnNoise, &m.ConnSNR, &m.ConnChannel, &m.ConnBand, &m.ConnLinkRate, &m.ConnDuplex,
		); err != nil {
			return nil, err
		}
		if lat.Valid {
			m.Latency = lat.Float64
		}
		if jit.Valid {
			m.Jitter = jit.Float64
		}
		if loss.Valid {
			m.PacketLoss = loss.Float64
		}
		if dl.Valid {
			m.Download = dl.Float64
		}
		if ul.Valid {
			m.Upload = ul.Float64
		}
		if dns.Valid {
			m.DNS = dns.Float64
		}

		m.Timestamp = ts
		t, _ := time.Parse(time.RFC3339, ts)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02T15:04:05Z", ts)
		}
		m.Time = t.Format(timeFmt)
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []Measurement{}
	}
	return results, nil
}

// --- Summary ---

type Summary struct {
	LatencyAvg    float64 `json:"latency_avg"`
	LatencyP95    float64 `json:"latency_p95"`
	LatencyMin    float64 `json:"latency_min"`
	LatencyMax    float64 `json:"latency_max"`
	JitterAvg     float64 `json:"jitter_avg"`
	PacketLoss    float64 `json:"packet_loss"`
	DNSAvg        float64 `json:"dns_avg"`
	DownloadAvg   float64 `json:"download_avg"`
	UploadAvg     float64 `json:"upload_avg"`
	DownloadPeak  float64 `json:"download_peak"`
	UploadPeak    float64 `json:"upload_peak"`
	Uptime24h     float64 `json:"uptime_24h"`
	UptimePercent float64 `json:"uptime_percent"`
	UptimeScore   float64 `json:"uptime_score"`
	Outages24h    int     `json:"outages_24h"`
	FailedChecks  int     `json:"failed_checks"`
	TotalChecks   int     `json:"total_checks"`
}

func (s *Store) GetSummary() (Summary, error) {
	var sum Summary
	cutoff := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	err := s.db.QueryRow(q["get_summary_stats"], cutoff).Scan(
		&sum.LatencyAvg, &sum.LatencyMin, &sum.LatencyMax,
		&sum.JitterAvg, &sum.PacketLoss, &sum.DNSAvg,
	)
	if err != nil && err != sql.ErrNoRows {
		return sum, err
	}

	// Read actual speed tests summary
	_ = s.db.QueryRow(q["get_speed_summary_stats"], cutoff).Scan(
		&sum.DownloadAvg, &sum.UploadAvg, &sum.DownloadPeak, &sum.UploadPeak,
	)

	sum.LatencyAvg = round1(sum.LatencyAvg)
	sum.LatencyMin = round1(sum.LatencyMin)
	sum.LatencyMax = round1(sum.LatencyMax)
	sum.JitterAvg = round1(sum.JitterAvg)
	sum.PacketLoss = round1(sum.PacketLoss)
	sum.DNSAvg = round1(sum.DNSAvg)
	sum.DownloadAvg = round2(sum.DownloadAvg)
	sum.UploadAvg = round2(sum.UploadAvg)
	sum.DownloadPeak = round2(sum.DownloadPeak)
	sum.UploadPeak = round2(sum.UploadPeak)

	// Compute P95 excluding 0ms outage records
	rows, err := s.db.Query(q["get_latencies"], cutoff)
	if err == nil {
		defer rows.Close()
		var latencies []float64
		for rows.Next() {
			var l float64
			if err := rows.Scan(&l); err == nil && l > 0 {
				latencies = append(latencies, l)
			}
		}
		if len(latencies) > 0 {
			idx := int(float64(len(latencies)) * 0.95)
			if idx >= len(latencies) {
				idx = len(latencies) - 1
			}
			sum.LatencyP95 = round1(latencies[idx])
		}
	}

	// Calculate mathematically sound uptime & consecutive-cycle outages
	var total, up, failed int
	_ = s.db.QueryRow(q["get_summary_counts"], cutoff).Scan(&total, &up, &failed)
	sum.TotalChecks = total
	sum.FailedChecks = failed

	// Outage Definition: An outage requires >= 2 consecutive failed checks (packet_loss >= 100).
	// A single isolated dropped ping probe is transient loss, not a sustained network outage.
	outages := 0
	lossRows, lErr := s.db.Query(q["get_all_losses"], cutoff)
	if lErr == nil {
		defer lossRows.Close()
		consecutiveFails := 0
		for lossRows.Next() {
			var lossVal float64
			if err := lossRows.Scan(&lossVal); err == nil {
				if lossVal >= 99.9 {
					consecutiveFails++
					if consecutiveFails == 2 {
						outages++ // Transition into confirmed outage
					}
				} else {
					consecutiveFails = 0
				}
			}
		}
	}

	sum.Outages24h = outages

	if total > 0 {
		sum.Uptime24h = math.Round(float64(up)/float64(total)*10000) / 100
		sum.UptimePercent = sum.Uptime24h
		sUptime := 0.0
		switch {
		case sum.Uptime24h >= 100:
			sUptime = 100
		case sum.Uptime24h >= 99.0:
			sUptime = 95.0 + ((sum.Uptime24h-99.0)/1.0)*5.0
		case sum.Uptime24h >= 95.0:
			sUptime = 80.0 + ((sum.Uptime24h-95.0)/4.0)*15.0
		case sum.Uptime24h >= 90.0:
			sUptime = 60.0 + ((sum.Uptime24h-90.0)/5.0)*20.0
		case sum.Uptime24h >= 80.0:
			sUptime = 40.0 + ((sum.Uptime24h-80.0)/10.0)*20.0
		case sum.Uptime24h >= 50.0:
			sUptime = 15.0 + ((sum.Uptime24h-50.0)/30.0)*25.0
		case sum.Uptime24h > 0:
			sUptime = (sum.Uptime24h / 50.0) * 15.0
		default:
			sUptime = 0
		}
		if outages > 0 {
			penalty := math.Min(15.0, float64(outages)*1.5)
			sUptime = math.Max(0, sUptime-penalty)
		}
		sum.UptimeScore = math.Max(0, math.Min(100, math.Round(sUptime)))
	} else {
		sum.Uptime24h = 100
		sum.UptimePercent = 100
		sum.UptimeScore = 100
	}

	return sum, nil
}

// PruneOldRecords prunes historical measurements, speed tests, and network events older than retentionDays.
func (s *Store) PruneOldRecords(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Format(time.RFC3339)
	if _, err := s.db.Exec(q["prune_measurements"], cutoff); err != nil {
		return err
	}
	if _, err := s.db.Exec(q["prune_speed_tests"], cutoff); err != nil {
		return err
	}
	if _, err := s.db.Exec(q["prune_network_events"], cutoff); err != nil {
		return err
	}
	return nil
}

// --- PingTarget ---

type PingTarget struct {
	Host        string  `json:"host"`
	IP          string  `json:"ip"`
	Latency     float64 `json:"latency"`
	MinLatency  float64 `json:"min_latency"`
	MaxLatency  float64 `json:"max_latency"`
	AvgLatency  float64 `json:"avg_latency"`
	Jitter      float64 `json:"jitter"`
	Loss        float64 `json:"loss"`
	ProbesSent  int     `json:"probes_sent"`
	ProbesRecv  int     `json:"probes_recv"`
	ProbesLost  int     `json:"probes_lost"`
	Method      string  `json:"method"` // "icmp", "tcp_fallback"
	Status      string  `json:"status"`
	LastChecked string  `json:"last_checked"`
}

func (s *Store) UpsertPingTarget(t PingTarget) error {
	method := t.Method
	if method == "" {
		method = "icmp"
	}
	probesSent := t.ProbesSent
	if probesSent == 0 {
		probesSent = 1
	}
	_, err := s.db.Exec(q["upsert_ping_target"],
		t.Host, t.IP, t.Latency, t.MinLatency, t.MaxLatency, t.AvgLatency, t.Jitter, t.Loss,
		probesSent, t.ProbesRecv, t.ProbesLost, method, t.Status, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetPingTargets() ([]PingTarget, error) {
	rows, err := s.db.Query(q["get_ping_targets"])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := []PingTarget{}
	for rows.Next() {
		var t PingTarget
		if err := rows.Scan(
			&t.Host, &t.IP, &t.Latency, &t.MinLatency, &t.MaxLatency, &t.AvgLatency, &t.Jitter, &t.Loss,
			&t.ProbesSent, &t.ProbesRecv, &t.ProbesLost, &t.Method, &t.Status, &t.LastChecked,
		); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

// --- DNSCheck ---

type DNSCheck struct {
	Host        string  `json:"host"`
	TimeMs      float64 `json:"time_ms"`
	Resolver    string  `json:"resolver"`
	ServerIP    string  `json:"server_ip"`
	Status      string  `json:"status"` // SUCCESS, TIMEOUT, NXDOMAIN, SERVFAIL, FAILED
	RCodeName   string  `json:"rcode_name"`
	Error       string  `json:"error,omitempty"`
	LastChecked string  `json:"last_checked"`
}

func (s *Store) UpsertDNSCheck(d DNSCheck) error {
	status := d.Status
	if status == "" {
		status = "SUCCESS"
	}
	rcodeName := d.RCodeName
	if rcodeName == "" {
		rcodeName = "NOERROR"
	}
	_, err := s.db.Exec(q["upsert_dns_check"],
		d.Host, d.TimeMs, d.Resolver, d.ServerIP, status, rcodeName, d.Error, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetDNSChecks() ([]DNSCheck, error) {
	rows, err := s.db.Query(q["get_dns_checks"])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := []DNSCheck{}
	for rows.Next() {
		var d DNSCheck
		if err := rows.Scan(
			&d.Host, &d.TimeMs, &d.Resolver, &d.ServerIP, &d.Status, &d.RCodeName, &d.Error, &d.LastChecked,
		); err != nil {
			return nil, err
		}
		checks = append(checks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return checks, nil
}

// --- NetworkEvent ---

type NetworkEvent struct {
	ID      int64  `json:"id"`
	Time    string `json:"time"`
	Network string `json:"network_id"`
	SSID    string `json:"ssid"`
	Gateway string `json:"gateway"`
}

func (s *Store) LogNetworkEvent(e NetworkEvent) error {
	_, err := s.db.Exec(q["log_network_event"], e.Time, e.Network, e.SSID, e.Gateway)
	return err
}

func (s *Store) GetRecentNetworkEvents(limit int) ([]NetworkEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(q["get_recent_network_events"], limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []NetworkEvent
	for rows.Next() {
		var e NetworkEvent
		if err := rows.Scan(&e.ID, &e.Time, &e.Network, &e.SSID, &e.Gateway); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if events == nil {
		events = []NetworkEvent{}
	}
	return events, nil
}

func (s *Store) GetCurrentNetworkID() string {
	var id string
	_ = s.db.QueryRow(q["get_current_network_id"]).Scan(&id)
	return id
}

// --- ConfigSettings ---

type ConfigSettings struct {
	PingTargets    []string `json:"ping_targets"`
	DNSTargets     []string `json:"dns_targets"`
	PingIntervalS  int      `json:"ping_interval_s"`
	SpeedIntervalM int      `json:"speed_interval_m"`
	PingCount      int      `json:"ping_count"`
}

func (s *Store) GetConfig() (ConfigSettings, error) {
	var raw string
	err := s.db.QueryRow(q["get_config"]).Scan(&raw)
	if err == sql.ErrNoRows {
		return ConfigSettings{}, sql.ErrNoRows
	}
	if err != nil {
		return ConfigSettings{}, err
	}
	var cs ConfigSettings
	return cs, json.Unmarshal([]byte(raw), &cs)
}

func (s *Store) SaveConfig(cs ConfigSettings) error {
	b, err := json.Marshal(cs)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(q["save_config"], string(b))
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
