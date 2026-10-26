/**
 * Connection Health Score Engine
 *
 * Provides a reproducible, documented 0-100 Connection Health Score
 * with normalized sub-scores and human-readable diagnostic explanations.
 */

export const DEFAULT_WEIGHTS = {
  loss: 0.30,
  latency: 0.25,
  jitter: 0.15,
  dns: 0.15,
  uptime: 0.15,
  speed: 0.10,
}

/**
 * Normalizes packet loss (%) to 0-100 score.
 */
export function scoreLoss(loss) {
  if (loss === undefined || loss === null || isNaN(loss) || loss < 0) return null
  if (loss === 0) return 100
  if (loss <= 1.0) return Math.round(100 - (loss / 1.0) * 15) // 100 -> 85
  if (loss <= 5.0) return Math.round(85 - ((loss - 1.0) / 4.0) * 45) // 85 -> 40
  if (loss <= 10.0) return Math.round(40 - ((loss - 5.0) / 5.0) * 40) // 40 -> 0
  return 0
}

/**
 * Normalizes latency (ms) to 0-100 score.
 */
export function scoreLatency(lat) {
  if (lat === undefined || lat === null || isNaN(lat) || lat <= 0) return null
  if (lat <= 20) return 100
  if (lat <= 50) return Math.round(100 - ((lat - 20) / 30) * 20) // 100 -> 80
  if (lat <= 150) return Math.round(80 - ((lat - 50) / 100) * 40) // 80 -> 40
  if (lat <= 300) return Math.round(40 - ((lat - 150) / 150) * 40) // 40 -> 0
  return 0
}

/**
 * Normalizes jitter (ms) to 0-100 score.
 */
export function scoreJitter(jit) {
  if (jit === undefined || jit === null || isNaN(jit) || jit < 0) return null
  if (jit <= 3) return 100
  if (jit <= 10) return Math.round(100 - ((jit - 3) / 7) * 25) // 100 -> 75
  if (jit <= 30) return Math.round(75 - ((jit - 10) / 20) * 45) // 75 -> 30
  return Math.max(0, Math.round(30 - (jit - 30)))
}

/**
 * Normalizes DNS lookup time (ms) to 0-100 score.
 */
export function scoreDNS(dns) {
  if (dns === undefined || dns === null || isNaN(dns) || dns <= 0) return null
  if (dns <= 15) return 100
  if (dns <= 50) return Math.round(100 - ((dns - 15) / 35) * 25) // 100 -> 75
  if (dns <= 150) return Math.round(75 - ((dns - 50) / 100) * 45) // 75 -> 30
  return Math.max(0, Math.round(30 - (dns - 150)))
}

/**
 * Normalizes uptime / availability to 0-100 score.
 */
export function scoreUptime(uptimePct, outages = 0) {
  if (uptimePct === undefined || uptimePct === null || isNaN(uptimePct) || uptimePct < 0) {
    if (outages > 0) return Math.max(0, Math.round(100 - Math.min(30, outages * 3)))
    return null
  }

  let base = 0
  if (uptimePct >= 100) {
    base = 100
  } else if (uptimePct >= 99.0) {
    base = 95.0 + ((uptimePct - 99.0) / 1.0) * 5.0 // 99% -> 95, 100% -> 100
  } else if (uptimePct >= 95.0) {
    base = 80.0 + ((uptimePct - 95.0) / 4.0) * 15.0 // 95% -> 80, 99% -> 95
  } else if (uptimePct >= 90.0) {
    base = 60.0 + ((uptimePct - 90.0) / 5.0) * 20.0 // 90% -> 60, 95% -> 80
  } else if (uptimePct >= 80.0) {
    base = 40.0 + ((uptimePct - 80.0) / 10.0) * 20.0 // 80% -> 40, 90% -> 60
  } else if (uptimePct >= 50.0) {
    base = 15.0 + ((uptimePct - 50.0) / 30.0) * 25.0 // 50% -> 15, 80% -> 40
  } else if (uptimePct > 0) {
    base = (uptimePct / 50.0) * 15.0 // 0% -> 0, 50% -> 15
  } else {
    base = 0
  }

  if (outages > 0) {
    const penalty = Math.min(15.0, outages * 1.5)
    base = Math.max(0, base - penalty)
  }

  return Math.max(0, Math.min(100, Math.round(base)))
}

/**
 * Normalizes download speed (Mbps) to 0-100 score.
 */
export function scoreSpeed(dlMbps) {
  if (dlMbps === undefined || dlMbps === null || isNaN(dlMbps) || dlMbps <= 0) return null
  if (dlMbps >= 100) return 100
  if (dlMbps >= 50) return Math.round(85 + ((dlMbps - 50) / 50) * 15) // 85 -> 100
  if (dlMbps >= 25) return Math.round(70 + ((dlMbps - 25) / 25) * 15) // 70 -> 85
  if (dlMbps >= 10) return Math.round(50 + ((dlMbps - 10) / 15) * 20) // 50 -> 70
  return Math.max(0, Math.round(dlMbps * 5))
}

/**
 * Maps a 0-100 score to a human-readable grade and UI color token.
 */
export function rateScore(score) {
  if (score === null || score === undefined) return { label: '—', text: '—', color: 'var(--muted)', status: 'none' }
  if (score >= 85) return { label: 'GOOD', text: 'Good', color: 'var(--green)', status: 'good' }
  if (score >= 65) return { label: 'FAIR', text: 'Fair', color: 'var(--yellow)', status: 'fair' }
  if (score >= 40) return { label: 'WARNING', text: 'Warning', color: 'var(--yellow)', status: 'warn' }
  return { label: 'POOR', text: 'Poor', color: 'var(--red)', status: 'poor' }
}

/**
 * Calculates comprehensive Connection Health evaluation.
 */
export function evaluateHealth(summary, weights = DEFAULT_WEIGHTS) {
  if (!summary) {
    return {
      score: 0,
      rating: rateScore(0),
      components: {},
      reasons: ['No telemetry data available'],
      mainIssue: 'Waiting for network telemetry',
    }
  }

  // If 100% packet loss (total disconnect), health is 0
  if (summary.packet_loss >= 100) {
    return {
      score: 0,
      rating: rateScore(0),
      components: {
        loss: { name: 'Packet Loss', score: 0, raw: '100%', rating: rateScore(0) },
      },
      reasons: ['Connection offline: 100% packet loss'],
      mainIssue: 'Connection offline: 100% packet loss',
    }
  }

  const scores = {
    loss: scoreLoss(summary.packet_loss),
    latency: scoreLatency(summary.latency_avg),
    jitter: scoreJitter(summary.jitter_avg),
    dns: scoreDNS(summary.dns_avg),
    uptime: scoreUptime(summary.uptime_percent !== undefined ? summary.uptime_percent : summary.uptime_24h, summary.outages_24h),
    speed: scoreSpeed(summary.download_avg),
  }

  const rawValues = {
    loss: summary.packet_loss !== undefined ? `${summary.packet_loss}%` : null,
    latency: summary.latency_avg > 0 ? `${summary.latency_avg} ms` : null,
    jitter: summary.jitter_avg > 0 ? `${summary.jitter_avg} ms` : null,
    dns: summary.dns_avg > 0 ? `${summary.dns_avg} ms` : null,
    uptime: (summary.uptime_24h !== undefined && summary.uptime_24h >= 0) ? `${summary.uptime_24h}%` : null,
    speed: summary.download_avg > 0 ? `${summary.download_avg} Mbps` : null,
  }

  let weightedSum = 0
  let totalWeight = 0
  const components = {}
  const issues = []

  for (const [key, scoreVal] of Object.entries(scores)) {
    if (scoreVal !== null) {
      const w = weights[key] || 0.1
      weightedSum += scoreVal * w
      totalWeight += w
      const rating = rateScore(scoreVal)
      components[key] = {
        name: key.charAt(0).toUpperCase() + key.slice(1),
        score: scoreVal,
        raw: rawValues[key],
        rating,
      }

      // Collect diagnostic reasons
      if (scoreVal < 85) {
        let issueText = ''
        if (key === 'loss') issueText = `Packet loss at ${summary.packet_loss}%`
        else if (key === 'latency') issueText = `Latency elevated to ${summary.latency_avg} ms`
        else if (key === 'jitter') issueText = `Jitter elevated to ${summary.jitter_avg} ms`
        else if (key === 'dns') issueText = `DNS resolution slow (${summary.dns_avg} ms)`
        else if (key === 'uptime') issueText = `Uptime degraded (${summary.uptime_24h}%, ${summary.outages_24h} outages)`
        else if (key === 'speed') issueText = `Download speed limited (${summary.download_avg} Mbps)`
        issues.push({ key, score: scoreVal, text: issueText })
      }
    }
  }

  const finalScore = totalWeight > 0 ? Math.round(weightedSum / totalWeight) : 0
  const rating = rateScore(finalScore)

  // Sort issues by lowest score first
  issues.sort((a, b) => a.score - b.score)
  const reasons = issues.map(i => i.text)
  const mainIssue = issues.length > 0 ? issues[0].text : 'All network metrics within optimal thresholds'

  return {
    score: finalScore,
    rating,
    components,
    reasons,
    mainIssue,
  }
}

/**
 * Legacy compatibility helper.
 */
export function computeHealth(summary) {
  return evaluateHealth(summary).score
}

export function rateMetric(val, good, warn) {
  if (val <= good) return { text: 'Good', color: 'var(--green)' }
  if (val <= warn) return { text: 'Fair', color: 'var(--yellow)' }
  return { text: 'Poor', color: 'var(--red)' }
}
