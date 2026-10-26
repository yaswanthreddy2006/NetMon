-- name: insert_measurement
INSERT INTO measurements
	(ts, network_id, latency, jitter, packet_loss, download, upload, dns,
	 conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: insert_speed_test
INSERT INTO speed_tests
	(ts, download_mbps, upload_mbps, duration_seconds, bytes_down, bytes_up, status, error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: get_speed_tests_window
SELECT id, ts, download_mbps, upload_mbps, duration_seconds, bytes_down, bytes_up, status, error
FROM speed_tests WHERE ts >= ? ORDER BY ts ASC;

-- name: get_latest_speed_test
SELECT id, ts, download_mbps, upload_mbps, duration_seconds, bytes_down, bytes_up, status, error
FROM speed_tests ORDER BY ts DESC LIMIT 1;

-- name: get_speed_summary_stats
SELECT
	COALESCE(AVG(download_mbps), 0),
	COALESCE(AVG(upload_mbps),   0),
	COALESCE(MAX(download_mbps), 0),
	COALESCE(MAX(upload_mbps),   0)
FROM speed_tests WHERE ts >= ? AND status = 'ok' AND (download_mbps > 0 OR upload_mbps > 0);

-- name: get_history
SELECT id, ts, network_id, latency, jitter, packet_loss, download, upload, dns,
       conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex
FROM measurements ORDER BY ts DESC LIMIT ?;

-- name: get_history_window
SELECT id, ts, network_id, latency, jitter, packet_loss, download, upload, dns,
       conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex
FROM measurements WHERE ts >= ? ORDER BY ts ASC;

-- name: get_history_downsampled_5m
SELECT
	MIN(id) AS id,
	strftime('%Y-%m-%dT%H:%M:00Z', datetime((strftime('%s', ts) / 300) * 300, 'unixepoch')) AS ts,
	network_id,
	ROUND(AVG(CASE WHEN packet_loss < 100 AND latency > 0 THEN latency ELSE NULL END), 1) AS latency,
	ROUND(AVG(jitter), 1) AS jitter,
	ROUND(AVG(packet_loss), 1) AS packet_loss,
	ROUND(AVG(CASE WHEN download > 0 THEN download ELSE NULL END), 1) AS download,
	ROUND(AVG(CASE WHEN upload > 0 THEN upload ELSE NULL END), 1) AS upload,
	ROUND(AVG(CASE WHEN dns > 0 THEN dns ELSE NULL END), 1) AS dns,
	conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex
FROM measurements
WHERE ts >= ?
GROUP BY (strftime('%s', ts) / 300)
ORDER BY ts ASC;

-- name: get_history_downsampled_1h
SELECT
	MIN(id) AS id,
	strftime('%Y-%m-%dT%H:00:00Z', ts) AS ts,
	network_id,
	ROUND(AVG(CASE WHEN packet_loss < 100 AND latency > 0 THEN latency ELSE NULL END), 1) AS latency,
	ROUND(AVG(jitter), 1) AS jitter,
	ROUND(AVG(packet_loss), 1) AS packet_loss,
	ROUND(AVG(CASE WHEN download > 0 THEN download ELSE NULL END), 1) AS download,
	ROUND(AVG(CASE WHEN upload > 0 THEN upload ELSE NULL END), 1) AS upload,
	ROUND(AVG(CASE WHEN dns > 0 THEN dns ELSE NULL END), 1) AS dns,
	conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex
FROM measurements
WHERE ts >= ?
GROUP BY strftime('%Y-%m-%dT%H:00:00Z', ts)
ORDER BY ts ASC;

-- name: get_summary_stats
SELECT
	COALESCE(AVG(CASE WHEN packet_loss < 100 AND latency > 0 THEN latency ELSE NULL END), 0),
	COALESCE(MIN(CASE WHEN packet_loss < 100 AND latency > 0 THEN latency ELSE NULL END), 0),
	COALESCE(MAX(CASE WHEN packet_loss < 100 AND latency > 0 THEN latency ELSE NULL END), 0),
	COALESCE(AVG(jitter), 0),
	COALESCE(AVG(packet_loss), 0),
	COALESCE(AVG(CASE WHEN dns > 0 THEN dns ELSE NULL END), 0)
FROM measurements WHERE ts >= ?;

-- name: get_summary_counts
SELECT
	COUNT(*),
	COUNT(CASE WHEN packet_loss < 100 THEN 1 END),
	COUNT(CASE WHEN packet_loss >= 100 THEN 1 END)
FROM measurements WHERE ts >= ?;

-- name: get_latencies
SELECT latency FROM measurements WHERE ts >= ? AND packet_loss < 100 AND latency > 0 ORDER BY latency;

-- name: get_all_losses
SELECT packet_loss FROM measurements WHERE ts >= ? ORDER BY ts ASC;

-- name: prune_measurements
DELETE FROM measurements WHERE ts < ?;

-- name: prune_speed_tests
DELETE FROM speed_tests WHERE ts < ?;

-- name: prune_network_events
DELETE FROM network_events WHERE ts < ?;

-- name: upsert_ping_target
INSERT INTO ping_targets (host, ip, latency, min_latency, max_latency, avg_latency, jitter, loss, probes_sent, probes_recv, probes_lost, method, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(host) DO UPDATE SET
	ip          = excluded.ip,
	latency     = excluded.latency,
	min_latency = excluded.min_latency,
	max_latency = excluded.max_latency,
	avg_latency = excluded.avg_latency,
	jitter      = excluded.jitter,
	loss        = excluded.loss,
	probes_sent = excluded.probes_sent,
	probes_recv = excluded.probes_recv,
	probes_lost = excluded.probes_lost,
	method      = excluded.method,
	status      = excluded.status,
	updated_at  = excluded.updated_at;

-- name: get_ping_targets
SELECT host, ip, latency, min_latency, max_latency, avg_latency, jitter, loss, probes_sent, probes_recv, probes_lost, method, status, updated_at
FROM ping_targets ORDER BY host;

-- name: upsert_dns_check
INSERT INTO dns_checks (host, time_ms, resolver, server_ip, status, rcode_name, error, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(host) DO UPDATE SET
	time_ms    = excluded.time_ms,
	resolver   = excluded.resolver,
	server_ip  = excluded.server_ip,
	status     = excluded.status,
	rcode_name = excluded.rcode_name,
	error      = excluded.error,
	updated_at = excluded.updated_at;

-- name: get_dns_checks
SELECT host, time_ms, resolver, server_ip, status, rcode_name, error, updated_at
FROM dns_checks ORDER BY host;

-- name: get_latest_measurement
SELECT id, ts, network_id, latency, jitter, packet_loss, download, upload, dns,
       conn_type, conn_rssi, conn_noise, conn_snr, conn_channel, conn_band, conn_link_rate, conn_duplex
FROM measurements ORDER BY ts DESC LIMIT 1;

-- name: log_network_event
INSERT INTO network_events (ts, network_id, ssid, gateway) VALUES (?, ?, ?, ?);

-- name: get_recent_network_events
SELECT id, ts, network_id, ssid, gateway FROM network_events ORDER BY ts DESC LIMIT ?;

-- name: get_current_network_id
SELECT network_id FROM network_events ORDER BY ts DESC LIMIT 1;

-- name: get_config
SELECT value FROM settings WHERE key = 'config';

-- name: save_config
INSERT INTO settings (key, value) VALUES ('config', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;
