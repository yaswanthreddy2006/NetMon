package network

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

// DNSServerInfo contains discovered DNS resolver metadata.
type DNSServerInfo struct {
	IP        string `json:"ip"`
	Interface string `json:"interface,omitempty"`
	IsDefault bool   `json:"is_default"`
	Source    string `json:"source"`
}

// DetectSystemDNSServers queries the local operating system to discover configured DNS resolvers.
func DetectSystemDNSServers() []string {
	switch runtime.GOOS {
	case "windows":
		return detectDNSWindows()
	case "linux":
		return detectDNSLinux()
	case "darwin":
		return detectDNSDarwin()
	default:
		return detectDNSFallback()
	}
}

func detectDNSWindows() []string {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "interface", "ipv4", "show", "dnsservers")
	if err != nil {
		return detectDNSFallback()
	}

	var servers []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Lines like: "DNS servers configured through DHCP:  192.168.1.1" or "Statically Configured DNS Servers: 8.8.8.8" or "1.1.1.1"
		parts := strings.Split(line, ":")
		var candidate string
		if len(parts) >= 2 {
			candidate = strings.TrimSpace(parts[len(parts)-1])
		} else {
			candidate = line
		}

		ip := net.ParseIP(candidate)
		if ip != nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			ipStr := ip.String()
			if !seen[ipStr] {
				seen[ipStr] = true
				servers = append(servers, ipStr)
			}
		}
	}

	if len(servers) > 0 {
		return servers
	}
	return detectDNSFallback()
}

func detectDNSLinux() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return detectDNSFallback()
	}

	var servers []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ip := net.ParseIP(fields[1])
				if ip != nil && !ip.IsUnspecified() {
					ipStr := ip.String()
					if !seen[ipStr] {
						seen[ipStr] = true
						servers = append(servers, ipStr)
					}
				}
			}
		}
	}

	if len(servers) > 0 {
		return servers
	}
	return detectDNSFallback()
}

func detectDNSDarwin() []string {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "scutil", "--dns")
	if err != nil {
		return detectDNSLinux() // fallback to /etc/resolv.conf
	}

	var servers []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver[") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				cand := strings.TrimSpace(parts[1])
				ip := net.ParseIP(cand)
				if ip != nil && !ip.IsUnspecified() {
					ipStr := ip.String()
					if !seen[ipStr] {
						seen[ipStr] = true
						servers = append(servers, ipStr)
					}
				}
			}
		}
	}

	if len(servers) > 0 {
		return servers
	}
	return detectDNSFallback()
}

func detectDNSFallback() []string {
	// If system resolver cannot be enumerated from CLI, use gateway if available
	gw := DefaultGateway()
	if gw != "" && net.ParseIP(gw) != nil {
		return []string{gw}
	}
	return []string{"1.1.1.1", "8.8.8.8"}
}

// BuildDNSQuery creates an RFC 1035 wire-format DNS A-record query with a unique transaction ID.
func BuildDNSQuery(host string, id uint16) []byte {
	var buf []byte
	// 12-byte header
	hdr := make([]byte, 12)
	binary.BigEndian.PutUint16(hdr[0:2], id)
	hdr[2] = 0x01 // RD (recursion desired) = 1
	hdr[3] = 0x00
	binary.BigEndian.PutUint16(hdr[4:6], 1) // QDCOUNT = 1
	buf = append(buf, hdr...)

	// Question: Name encoded as length-prefixed labels
	cleanHost := strings.Trim(host, ".")
	if cleanHost != "" {
		labels := strings.Split(cleanHost, ".")
		for _, l := range labels {
			if len(l) == 0 || len(l) > 63 {
				continue
			}
			buf = append(buf, byte(len(l)))
			buf = append(buf, []byte(l)...)
		}
	}
	buf = append(buf, 0x00) // root label terminator

	// QTYPE = 1 (A record)
	qtype := make([]byte, 2)
	binary.BigEndian.PutUint16(qtype, 1)
	buf = append(buf, qtype...)

	// QCLASS = 1 (IN)
	qclass := make([]byte, 2)
	binary.BigEndian.PutUint16(qclass, 1)
	buf = append(buf, qclass...)

	return buf
}

// DNSLookupResult holds detailed measurement and data quality status for a DNS test.
type DNSLookupResult struct {
	Host        string              `json:"host"`
	Resolver    string              `json:"resolver"`
	ServerIP    string              `json:"server_ip"`
	DurationMs  float64             `json:"duration_ms"`
	Status      MetricQualityStatus `json:"status"`
	RCode       int                 `json:"rcode"`
	RCodeName   string              `json:"rcode_name"`
	Error       string              `json:"error,omitempty"`
	LastChecked string              `json:"last_checked"`
}

// MeasureDNSQuery benchmarks direct UDP query to an authoritative/recursive resolver without OS cache skew.
func MeasureDNSQuery(ctx context.Context, host, resolverAddr string) DNSLookupResult {
	nowRFC := time.Now().Format(time.RFC3339)
	targetResolver := resolverAddr
	if targetResolver == "" || targetResolver == "system" {
		sysServers := DetectSystemDNSServers()
		if len(sysServers) > 0 {
			targetResolver = sysServers[0]
		} else {
			targetResolver = "1.1.1.1"
		}
	}

	dialAddr := targetResolver
	if !strings.Contains(dialAddr, ":") {
		dialAddr += ":53"
	}

	res := DNSLookupResult{
		Host:        host,
		Resolver:    resolverAddr,
		ServerIP:    targetResolver,
		LastChecked: nowRFC,
	}

	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "udp", dialAddr)
	if err != nil {
		res.Status = QualityFailed
		res.Error = fmt.Sprintf("dial error: %v", err)
		return res
	}
	defer conn.Close()

	deadline := time.Now().Add(3 * time.Second)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)

	txID := uint16(rand.Intn(65535))
	query := BuildDNSQuery(host, txID)

	start := time.Now()
	if _, err := conn.Write(query); err != nil {
		res.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0
		res.Status = QualityFailed
		res.Error = fmt.Sprintf("write error: %v", err)
		return res
	}

	resp := make([]byte, 512)
	n, err := conn.Read(resp)
	elapsed := time.Since(start)
	res.DurationMs = float64(elapsed.Microseconds()) / 1000.0

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			res.Status = QualityTimeout
			res.Error = "DNS query timed out (3000ms)"
		} else {
			res.Status = QualityFailed
			res.Error = fmt.Sprintf("read error: %v", err)
		}
		return res
	}

	if n < 12 {
		res.Status = QualityFailed
		res.Error = fmt.Sprintf("truncated response (%d bytes)", n)
		return res
	}

	respID := binary.BigEndian.Uint16(resp[0:2])
	if respID != txID {
		res.Status = QualityFailed
		res.Error = fmt.Sprintf("transaction ID mismatch (got %d, expected %d)", respID, txID)
		return res
	}

	rcode := int(resp[3] & 0x0F)
	res.RCode = rcode

	switch rcode {
	case 0:
		res.Status = QualitySuccess
		res.RCodeName = "NOERROR"
	case 1:
		res.Status = QualityFailed
		res.RCodeName = "FORMERR"
		res.Error = "Format error in query"
	case 2:
		res.Status = QualityServFail
		res.RCodeName = "SERVFAIL"
		res.Error = "Server failure"
	case 3:
		res.Status = QualityNxDomain
		res.RCodeName = "NXDOMAIN"
		res.Error = "Non-existent domain"
	case 4:
		res.Status = QualityFailed
		res.RCodeName = "NOTIMP"
		res.Error = "Query type not implemented by server"
	case 5:
		res.Status = QualityFailed
		res.RCodeName = "REFUSED"
		res.Error = "Query refused by server"
	default:
		res.Status = QualityFailed
		res.RCodeName = fmt.Sprintf("RCODE_%d", rcode)
		res.Error = fmt.Sprintf("DNS error rcode %d", rcode)
	}

	return res
}
