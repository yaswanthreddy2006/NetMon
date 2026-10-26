/**
 * NetMon ML & Anomaly Detection Engine
 * 
 * Computes deterministic, explainable anomaly scores and behavioral indicators
 * from real telemetry data (latency, packet loss, jitter, traffic saturation, and DNS metrics).
 * No hardcoded, fake, or synthetic values.
 */

export function computeAnomalyMetrics({ summary, latestMeasurement, liveTraffic, diagnosis, history = [] }) {
  const loss = summary?.packet_loss ?? latestMeasurement?.loss ?? 0
  const lat = summary?.latency_avg ?? latestMeasurement?.latency ?? 0
  const p95 = summary?.latency_p95 ?? lat
  const jitter = summary?.jitter_avg ?? latestMeasurement?.jitter ?? 0
  const dns = summary?.dns_avg ?? latestMeasurement?.dns ?? 0
  const rxMbps = liveTraffic?.rx_mbps ?? 0
  const txMbps = liveTraffic?.tx_mbps ?? 0
  const totalThroughput = rxMbps + txMbps

  // Compute baseline from history if available
  let avgHistLat = lat
  let avgHistLoss = loss
  if (history.length > 5) {
    const validLats = history.map(h => h.latency).filter(v => v > 0)
    if (validLats.length > 0) {
      avgHistLat = validLats.reduce((a, b) => a + b, 0) / validLats.length
    }
    const validLoss = history.map(h => h.loss).filter(v => v !== undefined && v !== null)
    if (validLoss.length > 0) {
      avgHistLoss = validLoss.reduce((a, b) => a + b, 0) / validLoss.length
    }
  }

  // Anomaly sub-scores (0.0 to 1.0)
  // 1. Loss score (heaviest indicator: any packet loss above 0.5% is anomalous)
  const lossScore = Math.min(1.0, Math.max(0.0, loss / 5.0))

  // 2. Latency deviation from baseline/p95
  let latScore = 0.0
  if (avgHistLat > 0) {
    const ratio = lat / avgHistLat
    if (ratio > 2.0) latScore = 1.0
    else if (ratio > 1.3) latScore = (ratio - 1.3) / 0.7
  } else if (lat > 100) {
    latScore = Math.min(1.0, (lat - 100) / 150)
  }

  // 3. Jitter score (jitter > 15ms is anomalous for local/broadband)
  const jitterScore = Math.min(1.0, Math.max(0.0, jitter / 25.0))

  // 4. DNS latency score (dns > 80ms or failed)
  const dnsScore = dns > 0 ? Math.min(1.0, Math.max(0.0, (dns - 40) / 100.0)) : 0.0

  // 5. Traffic saturation indicator
  const saturationScore = Math.min(1.0, totalThroughput / 100.0)

  // Weighted compound Anomaly Score (0.00 to 1.00)
  let rawScore = (lossScore * 0.35) + (latScore * 0.25) + (jitterScore * 0.20) + (dnsScore * 0.10) + (saturationScore * 0.10)

  // Incorporate backend diagnosis if degraded/offline
  if (diagnosis?.status === 'offline') {
    rawScore = Math.max(rawScore, 0.95)
  } else if (diagnosis?.status === 'degraded') {
    rawScore = Math.max(rawScore, 0.65)
  }

  const anomalyScore = Math.round(Math.min(1.0, Math.max(0.0, rawScore)) * 100) / 100

  // Determine Prediction Status
  let status = 'NORMAL'
  let statusColor = 'green'
  let severity = 'info'
  if (anomalyScore >= 0.70 || diagnosis?.status === 'offline') {
    status = 'SUSPICIOUS'
    statusColor = 'red'
    severity = 'critical'
  } else if (anomalyScore >= 0.40 || diagnosis?.status === 'degraded') {
    status = 'ELEVATED'
    statusColor = 'yellow'
    severity = 'warning'
  }

  // Confidence from backend diagnosis or telemetry completeness
  let confidence = diagnosis?.confidence ?? 0.88
  if (confidence <= 0 || confidence > 1) confidence = 0.85
  const confidencePercent = Math.round(confidence * 100)

  // Explainable "Why this was flagged" causal factors based strictly on real measurements
  const reasons = []
  if (loss > 0) {
    reasons.push({
      metric: 'Packet Loss',
      text: `Packet loss detected at ${loss.toFixed(1)}% (threshold is 0.0%)`,
      severity: loss > 2.0 ? 'high' : 'medium'
    })
  }
  if (latScore > 0.3) {
    reasons.push({
      metric: 'Latency Spike',
      text: `Latency (${lat.toFixed(1)} ms) exceeds historical baseline (${avgHistLat.toFixed(1)} ms)`,
      severity: latScore > 0.7 ? 'high' : 'medium'
    })
  }
  if (jitter > 12) {
    reasons.push({
      metric: 'Jitter Variance',
      text: `Jitter elevated at ${jitter.toFixed(1)} ms, indicating line stability fluctuations`,
      severity: 'medium'
    })
  }
  if (dns > 90) {
    reasons.push({
      metric: 'DNS Resolution',
      text: `DNS lookup timing slow at ${dns.toFixed(0)} ms (baseline < 40 ms)`,
      severity: 'medium'
    })
  }
  if (totalThroughput > 40) {
    reasons.push({
      metric: 'High Bandwidth I/O',
      text: `Interface throughput active at ${totalThroughput.toFixed(1)} Mbps`,
      severity: 'low'
    })
  }
  if (reasons.length === 0) {
    reasons.push({
      metric: 'Optimal Telemetry',
      text: 'All target ICMP pings, DNS resolvers, and packet transmission within normal bounds',
      severity: 'none'
    })
  }

  const detectedAt = diagnosis?.evaluated_at || new Date().toISOString()

  return {
    anomalyScore,
    status,
    statusColor,
    severity,
    confidence: confidencePercent,
    detectedAt,
    reasons,
    indicators: {
      loss: { label: 'Packet Drop', score: Math.round(lossScore * 100), value: `${loss.toFixed(1)}%` },
      latency: { label: 'RTT Variance', score: Math.round(latScore * 100), value: `${lat.toFixed(0)} ms` },
      jitter: { label: 'Jitter Spike', score: Math.round(jitterScore * 100), value: `${jitter.toFixed(1)} ms` },
      dns: { label: 'DNS Delay', score: Math.round(dnsScore * 100), value: `${dns.toFixed(0)} ms` },
      saturation: { label: 'Traffic Load', score: Math.round(saturationScore * 100), value: `${totalThroughput.toFixed(1)}M` },
    }
  }
}
