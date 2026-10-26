/**
 * Time utility module for RFC3339 / ISO timestamps, relative elapsed time,
 * and stale-data detection.
 */

/**
 * Parses an ISO 8601 / RFC 3339 string or number into epoch milliseconds.
 * Returns null if the timestamp is missing or invalid.
 */
export function parseTimestamp(ts) {
  if (!ts) return null
  if (typeof ts === 'number') {
    // If it's in seconds (e.g. 10 digits), convert to milliseconds
    return ts < 1e11 ? ts * 1000 : ts
  }
  const parsed = new Date(ts).getTime()
  return isNaN(parsed) ? null : parsed
}

/**
 * Returns human-readable relative elapsed time string from a timestamp.
 * Handles negative deltas (future/clock-skew) gracefully by returning 'just now'.
 */
export function formatTimeAgo(ts, now = Date.now()) {
  const t = parseTimestamp(ts)
  if (t === null) return ''
  const diffSec = Math.floor((now - t) / 1000)
  if (diffSec < 5) return 'just now'
  if (diffSec < 60) return `${diffSec}s ago`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`
  return `${Math.floor(diffSec / 86400)}d ago`
}

/**
 * Checks whether a timestamp exceeds the configured staleness threshold.
 * If ts is null or empty, it is considered stale.
 */
export function isStale(ts, thresholdMs, now = Date.now()) {
  const t = parseTimestamp(ts)
  if (t === null) return true
  return (now - t) > thresholdMs
}

/**
 * Formats an exact timestamp to local time string.
 */
export function formatExactTime(ts, includeDate = false) {
  const t = parseTimestamp(ts)
  if (t === null) return '—'
  const d = new Date(t)
  if (includeDate) {
    return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
  }
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
