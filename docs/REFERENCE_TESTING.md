# NetMon Accuracy & Reference Testing Guide

This document details the technical benchmarking methodology for validating NetMon's measurements against reference operating system tools and third-party network utilities.

---

## 1. Reference Comparison Methodology

When comparing NetMon against external reference tools, avoid expecting exact decimal identity across single instantaneous probes because network path latency, congestion, and queueing vary millisecond by millisecond. Instead, perform **at least 5 repeated samples** under identical conditions and evaluate the mean and percentage variance.

### Formula for Percentage Difference:
$$\text{percentage\_difference} = \frac{|\text{netmon} - \text{reference}|}{\text{reference}} \times 100$$

---

## 2. Latency & ICMP Probing

### Reference Tool:
- **Windows**: `ping -n 5 <target>`
- **Linux / macOS**: `ping -c 5 <target>`

### Comparison Attributes:
| Metric | NetMon Method | Reference OS Tool | Expected Variance |
| :--- | :--- | :--- | :--- |
| **Gateway RTT** | High-resolution monotonic timer with OS ping | `ping <gateway_ip>` | $\le 5\%$ |
| **Internet RTT** | Multi-sample ICMP Echo with sub-ms parsing | `ping 1.1.1.1` | $\le 10\%$ |
| **Packet Loss** | $\frac{\text{lost\_probes}}{\text{total\_probes}} \times 100$ | Summary loss line | Identical ($0\%$) |
| **Jitter** | RFC 3393 Consecutive IPDV | `ping` stddev / mdev | Close approximation |

### Fallback Behavior:
If ICMP packets are restricted or blocked by firewalls, NetMon falls back to TCP SYN connect probes (ports 443/80/53) and explicitly tags the measurement method as `tcp_fallback`.

---

## 3. DNS Lookup Timing

### Reference Tool:
- **Command**: `nslookup <hostname> <server_ip>` or `dig @<server_ip> <hostname>`

### Comparison Attributes:
| Metric | NetMon Method | Reference Tool | Notes |
| :--- | :--- | :--- | :--- |
| **Resolution Time** | Direct UDP query with random TxID (RFC 1035) | `dig` `query time` | Bypasses local OS DNS cache |
| **Error Reporting** | RCODE detection (NOERROR, NXDOMAIN, SERVFAIL) | Return code | Explicit error code retained |

> [!NOTE]
> NetMon's direct UDP benchmarking measures raw network round-trip time to the designated resolver without local OS client cache skew.

---

## 4. Wi-Fi Signal & Physical Layer Telemetry

### Reference Tool:
- **Windows**: `netsh wlan show interfaces`
- **Linux**: `iw dev <iface> link`
- **macOS**: `system_profiler SPAirPortDataType -json`

### Metric Definitions:
- **RSSI**: Native Received Signal Strength Indicator in dBm (negative number; e.g. -47 dBm).
- **Wi-Fi PHY Link Rate**: The negotiated 802.11 physical radio link capacity (e.g., 1201 Mbps on 802.11ax 2x2 80MHz).
  > [!IMPORTANT]
  > **Wi-Fi PHY Link Rate is NEVER labeled "Internet Speed"**. It represents the local link capacity between the device and the Wi-Fi access point, not upstream ISP throughput.

---

## 5. Internet Throughput (Download & Upload)

### Reference Services:
- Fast.com (Netflix CDN)
- Speedtest.net (Ookla)
- Cloudflare Speed Test

### Measurement Formula:
$$\text{throughput\_mbps} = \frac{\text{bytes\_transferred} \times 8}{\text{elapsed\_seconds} \times 1,000,000}$$
*(SI decimal Megabits per second, $10^6\text{ bits/s}$)*

### Factors Influencing Variance:
1. **Server Distance / Routing**: Different speed test endpoints (e.g. Cloudflare Edge vs regional Ookla host).
2. **Number of Concurrent Streams**: NetMon utilizes multi-stream transfers to saturate link capacity.
3. **Warm-up & First Byte Timing**: NetMon uses `httptrace.GotFirstResponseByte` to measure pure transfer duration excluding TCP/TLS handshake latency.
4. **Local Network & Wi-Fi Contention**: Other devices sharing the local access point.

---

## 6. Diagnostic Self-Test CLI

To run the automated internal verification suite at any time:

```bash
netmon --self-test
```

This verifies:
- `[Connectivity]`: Gateway detection, gateway latency, internet reachability, packet loss formula
- `[DNS]`: OS resolver detection, lookup timing, timeout handling, NXDOMAIN handling
- `[Wi-Fi]`: Native RSSI, Band, Channel, PHY Link Rate, and unsupported metric handling
- `[Throughput]`: Multi-stream transfers, byte counters, unit conversions, and timing accuracy
- `[Historical]`: SQLite sample persistence, 24-hour aggregation, consecutive outage detection
- `[Hotspot]`: Adapter discovery, client lifecycle, and bandwidth accounting
