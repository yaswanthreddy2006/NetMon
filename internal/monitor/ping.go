package monitor

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"netmon/internal/network"
)

type pingResult struct {
	Host       string
	IP         string
	Method     string  // "icmp", "tcp_fallback"
	ProbesSent int
	ProbesRecv int
	ProbesLost int
	MinMs      float64
	MaxMs      float64
	AvgMs      float64
	JitterMs   float64
	PacketLoss float64
	Status     string  // "up", "degraded", "down"
	Err        error
}

// runPing pings target count times with a context and returns avg RTT, RFC 3393 jitter, and packet loss.
// If ICMP is blocked/restricted by OS or firewall, it gracefully falls back to TCP SYN connect probes.
func runPing(ctx context.Context, target string, count int) (avg, jitter, loss float64, err error) {
	res := runDetailedPing(ctx, target, count)
	return res.AvgMs, res.JitterMs, res.PacketLoss, res.Err
}

// runDetailedPing executes high-accuracy probing with probe counting, min/max tracking, and TCP fallback.
func runDetailedPing(ctx context.Context, target string, count int) pingResult {
	if count < 1 {
		count = 1
	}

	ip := target
	if net.ParseIP(target) == nil {
		ip = resolveIP(ctx, target)
	}

	// 1. Attempt ICMP Ping via native OS tool
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), "-w", "2000", target)
	} else if runtime.GOOS == "darwin" {
		if os.Geteuid() == 0 {
			cmd = exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-i", "0.2", "-W", "2000", target)
		} else {
			cmd = exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2000", target)
		}
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2", target)
	}

	out, cmdErr := cmd.CombinedOutput()
	text := string(out)

	if ctx.Err() != nil {
		return pingResult{
			Host:       target,
			IP:         ip,
			Method:     "icmp",
			ProbesSent: count,
			ProbesRecv: 0,
			ProbesLost: count,
			PacketLoss: 100,
			Status:     "down",
			Err:        ctx.Err(),
		}
	}

	var avg, jitter, loss, minMs, maxMs float64
	var parseErr error
	var rtts []float64

	if runtime.GOOS == "windows" {
		avg, jitter, loss, rtts, parseErr = parsePingWindowsDetailed(text, count)
	} else {
		avg, jitter, loss, rtts, parseErr = parsePingUnixDetailed(text, count)
	}

	// If ICMP ping succeeded with at least one reply
	if parseErr == nil && len(rtts) > 0 {
		minMs = rtts[0]
		maxMs = rtts[0]
		for _, r := range rtts {
			if r < minMs {
				minMs = r
			}
			if r > maxMs {
				maxMs = r
			}
		}
		probesLost := count - len(rtts)
		if probesLost < 0 {
			probesLost = 0
		}
		status := "up"
		if loss > 0 && loss < 100 {
			status = "degraded"
		} else if loss >= 100 {
			status = "down"
		}

		return pingResult{
			Host:       target,
			IP:         ip,
			Method:     "icmp",
			ProbesSent: count,
			ProbesRecv: len(rtts),
			ProbesLost: probesLost,
			MinMs:      minMs,
			MaxMs:      maxMs,
			AvgMs:      avg,
			JitterMs:   jitter,
			PacketLoss: loss,
			Status:     status,
			Err:        nil,
		}
	}

	// 2. Check if we should use TCP connect fallback
	// If host could not be resolved or all ICMP packets timed out on internet hosts where ICMP is often firewalled
	if net.ParseIP(target) != nil || (ip != "" && !strings.Contains(strings.ToLower(text), "could not find host")) {
		tcpRes := probeTCPFallback(ctx, target, ip, count)
		if tcpRes.ProbesRecv > 0 {
			return tcpRes
		}
	}

	// Return ICMP failure result
	var finalErr error = parseErr
	if finalErr == nil && cmdErr != nil {
		finalErr = fmt.Errorf("ping command error: %w", cmdErr)
	}
	if finalErr == nil {
		finalErr = fmt.Errorf("100%% packet loss")
	}

	return pingResult{
		Host:       target,
		IP:         ip,
		Method:     "icmp",
		ProbesSent: count,
		ProbesRecv: 0,
		ProbesLost: count,
		PacketLoss: 100,
		Status:     "down",
		Err:        finalErr,
	}
}

// probeTCPFallback probes ports 443 / 80 / 53 to measure monotonic RTT when ICMP is blocked.
func probeTCPFallback(ctx context.Context, host, ip string, count int) pingResult {
	targetHost := host
	if ip != "" {
		targetHost = ip
	}

	ports := []string{"443", "80", "53"}
	var targetPort string
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(targetHost, p), 1*time.Second)
		if err == nil {
			conn.Close()
			targetPort = p
			break
		}
	}

	if targetPort == "" {
		return pingResult{
			Host:       host,
			IP:         ip,
			Method:     "tcp_fallback",
			ProbesSent: count,
			ProbesRecv: 0,
			ProbesLost: count,
			PacketLoss: 100,
			Status:     "down",
			Err:        fmt.Errorf("tcp fallback connect failed on ports 443/80/53"),
		}
	}

	var rtts []float64
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(targetHost, targetPort), 2*time.Second)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		if err == nil {
			conn.Close()
			rtts = append(rtts, elapsed)
		}
		if i < count-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	if len(rtts) == 0 {
		return pingResult{
			Host:       host,
			IP:         ip,
			Method:     "tcp_fallback",
			ProbesSent: count,
			ProbesRecv: 0,
			ProbesLost: count,
			PacketLoss: 100,
			Status:     "down",
			Err:        fmt.Errorf("tcp probes timed out"),
		}
	}

	var sum float64
	minMs := rtts[0]
	maxMs := rtts[0]
	for _, r := range rtts {
		sum += r
		if r < minMs {
			minMs = r
		}
		if r > maxMs {
			maxMs = r
		}
	}
	avg := sum / float64(len(rtts))
	jitter := calculateRFC3393Jitter(rtts)
	lost := count - len(rtts)
	if lost < 0 {
		lost = 0
	}
	loss := (float64(lost) / float64(count)) * 100

	status := "up"
	if loss > 0 && loss < 100 {
		status = "degraded"
	} else if loss >= 100 {
		status = "down"
	}

	return pingResult{
		Host:       host,
		IP:         ip,
		Method:     "tcp_fallback",
		ProbesSent: count,
		ProbesRecv: len(rtts),
		ProbesLost: lost,
		MinMs:      minMs,
		MaxMs:      maxMs,
		AvgMs:      avg,
		JitterMs:   jitter,
		PacketLoss: loss,
		Status:     status,
		Err:        nil,
	}
}

// calculateRFC3393Jitter computes Consecutive Packet Delay Variation (IPDV)
// Jitter = sum(|RTT[i+1] - RTT[i]|) / (N - 1)
func calculateRFC3393Jitter(rtts []float64) float64 {
	if len(rtts) < 2 {
		return 0
	}
	var sumDiff float64
	for i := 0; i < len(rtts)-1; i++ {
		sumDiff += math.Abs(rtts[i+1] - rtts[i])
	}
	return sumDiff / float64(len(rtts)-1)
}

func parsePingWindows(text string, expectedCount int) (avg, jitter, loss float64, err error) {
	avg, jitter, loss, _, err = parsePingWindowsDetailed(text, expectedCount)
	return avg, jitter, loss, err
}

func parsePingWindowsDetailed(text string, expectedCount int) (avg, jitter, loss float64, rtts []float64, err error) {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "could not find host") ||
		strings.Contains(lower, "not find host") ||
		strings.Contains(lower, "nicht finden") ||
		strings.Contains(lower, "introuvable") ||
		strings.Contains(lower, "no se pudo encontrar") {
		return 0, 0, 100, nil, fmt.Errorf("ping: could not resolve host")
	}

	replyRe := regexp.MustCompile(`(?i)(?:time|zeit|temps|tiempo)\s*([=<])\s*([\d.]+)\s*ms`)

	for _, line := range strings.Split(text, "\n") {
		if m := replyRe.FindStringSubmatch(line); m != nil {
			op := m[1]
			valStr := m[2]
			val, parseErr := strconv.ParseFloat(valStr, 64)
			if parseErr == nil {
				if op == "<" {
					val = 0.5
				}
				rtts = append(rtts, val)
			}
		}
	}

	lossRe := regexp.MustCompile(`\((\d+)%\s*(?:loss|Verlust|perte|perdidos|丢失)\)`)
	var parsedLoss float64
	hasSummaryLoss := false
	if m := lossRe.FindStringSubmatch(text); m != nil {
		if l, parseErr := strconv.ParseFloat(m[1], 64); parseErr == nil {
			parsedLoss = l
			hasSummaryLoss = true
		}
	}

	if len(rtts) == 0 {
		return 0, 0, 100, nil, fmt.Errorf("ping: all packets lost or destination unreachable")
	}

	var sumRtt float64
	for _, r := range rtts {
		sumRtt += r
	}
	avg = sumRtt / float64(len(rtts))
	jitter = calculateRFC3393Jitter(rtts)

	if hasSummaryLoss {
		loss = parsedLoss
	} else if expectedCount > 0 {
		lost := expectedCount - len(rtts)
		if lost < 0 {
			lost = 0
		}
		loss = (float64(lost) / float64(expectedCount)) * 100
	} else {
		loss = 0
	}

	return avg, jitter, loss, rtts, nil
}

func parsePingUnix(text string, expectedCount int) (avg, jitter, loss float64, err error) {
	avg, jitter, loss, _, err = parsePingUnixDetailed(text, expectedCount)
	return avg, jitter, loss, err
}

func parsePingUnixDetailed(text string, expectedCount int) (avg, jitter, loss float64, rtts []float64, err error) {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "unknown host") ||
		strings.Contains(lower, "cannot resolve") ||
		strings.Contains(lower, "name or service not known") {
		return 0, 0, 100, nil, fmt.Errorf("ping: could not resolve host")
	}

	sampleRe := regexp.MustCompile(`icmp_seq=\d+.*time=([\d.]+)\s*ms`)
	for _, line := range strings.Split(text, "\n") {
		if m := sampleRe.FindStringSubmatch(line); m != nil {
			if v, parseErr := strconv.ParseFloat(m[1], 64); parseErr == nil {
				rtts = append(rtts, v)
			}
		}
	}

	lossRe := regexp.MustCompile(`([\d.]+)%\s*(?:packet loss|loss)`)
	hasSummaryLoss := false
	if m := lossRe.FindStringSubmatch(text); m != nil {
		if l, parseErr := strconv.ParseFloat(m[1], 64); parseErr == nil {
			loss = l
			hasSummaryLoss = true
		}
	}

	rttRe := regexp.MustCompile(`min/avg/max/(?:\w+)\s*=\s*[\d.]+/([\d.]+)/[\d.]+/([\d.]+)`)
	var summaryAvg, summaryJitter float64
	hasSummaryRtt := false
	if m := rttRe.FindStringSubmatch(text); m != nil {
		summaryAvg, _ = strconv.ParseFloat(m[1], 64)
		summaryJitter, _ = strconv.ParseFloat(m[2], 64)
		hasSummaryRtt = true
	}

	if len(rtts) > 0 {
		var sumRtt float64
		for _, r := range rtts {
			sumRtt += r
		}
		avg = sumRtt / float64(len(rtts))
		jitter = calculateRFC3393Jitter(rtts)
	} else if hasSummaryRtt {
		avg = summaryAvg
		jitter = summaryJitter
	} else {
		return 0, 0, 100, nil, fmt.Errorf("ping: no response received")
	}

	if !hasSummaryLoss && expectedCount > 0 {
		lost := expectedCount - len(rtts)
		if lost < 0 {
			lost = 0
		}
		loss = (float64(lost) / float64(expectedCount)) * 100
	}

	if loss >= 100 {
		return 0, 0, 100, rtts, fmt.Errorf("ping: 100%% packet loss")
	}

	return avg, jitter, loss, rtts, nil
}

func resolveIP(ctx context.Context, host string) string {
	resolveCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(resolveCtx, "ip", host)
	if err != nil || len(ips) == 0 {
		return ""
	}
	return ips[0].String()
}

// measureDNS queries a DNS resolver directly over UDP to measure true network resolution time.
func measureDNS(ctx context.Context, host string, resolverAddr string) (time.Duration, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	res := network.MeasureDNSQuery(ctx, host, resolverAddr)
	if res.Status == network.QualitySuccess {
		return time.Duration(res.DurationMs * float64(time.Millisecond)), nil
	}
	if res.Error != "" {
		return time.Duration(res.DurationMs * float64(time.Millisecond)), fmt.Errorf("%s", res.Error)
	}
	return time.Duration(res.DurationMs * float64(time.Millisecond)), fmt.Errorf("dns check status: %s", res.Status)
}

// buildDNSQuery is a backward-compatibility wrapper for network.BuildDNSQuery.
var buildDNSQuery = network.BuildDNSQuery

