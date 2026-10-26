package monitor

import (
	"context"
	"math"
	"net"
	"testing"
	"time"
)

func TestCalculateRFC3393Jitter(t *testing.T) {
	// Example from requirement:
	// RTT: 10, 12, 11, 15
	// Differences: |12-10| + |11-12| + |15-11| = 2 + 1 + 4 = 7
	// N-1 = 3 -> 7/3 = 2.3333333333333335
	rtts := []float64{10, 12, 11, 15}
	expected := 7.0 / 3.0
	actual := calculateRFC3393Jitter(rtts)
	if math.Abs(actual-expected) > 1e-4 {
		t.Fatalf("expected jitter %f, got %f", expected, actual)
	}

	// Single sample -> 0 jitter
	if calculateRFC3393Jitter([]float64{10}) != 0 {
		t.Fatalf("expected 0 for single sample")
	}

	// Identical samples -> 0 jitter
	if calculateRFC3393Jitter([]float64{20, 20, 20, 20}) != 0 {
		t.Fatalf("expected 0 for identical samples")
	}
}

func TestParsePingWindows(t *testing.T) {
	// Standard Windows successful ping (English)
	windowsOutput := `
Pinging 8.8.8.8 with 32 bytes of data:
Reply from 8.8.8.8: bytes=32 time=10ms TTL=117
Reply from 8.8.8.8: bytes=32 time=12ms TTL=117
Reply from 8.8.8.8: bytes=32 time=11ms TTL=117
Reply from 8.8.8.8: bytes=32 time=15ms TTL=117

Ping statistics for 8.8.8.8:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
Approximate round trip times in milli-seconds:
    Minimum = 10ms, Maximum = 15ms, Average = 12ms
`
	avg, jitter, loss, err := parsePingWindows(windowsOutput, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loss != 0 {
		t.Fatalf("expected 0 loss, got %f", loss)
	}
	if avg != 12.0 {
		t.Fatalf("expected avg 12.0, got %f", avg)
	}
	expectedJitter := 7.0 / 3.0
	if math.Abs(jitter-expectedJitter) > 1e-4 {
		t.Fatalf("expected jitter %f, got %f", expectedJitter, jitter)
	}

	// Sub-millisecond Windows ping (<1ms on local gateway)
	windowsSubMs := `
Pinging 192.168.1.1 with 32 bytes of data:
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64
Reply from 192.168.1.1: bytes=32 time=1ms TTL=64
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64

Ping statistics for 192.168.1.1:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
Approximate round trip times in milli-seconds:
    Minimum = 0ms, Maximum = 1ms, Average = 0ms
`
	avgSub, jitterSub, lossSub, errSub := parsePingWindows(windowsSubMs, 4)
	if errSub != nil {
		t.Fatalf("unexpected error: %v", errSub)
	}
	if lossSub != 0 {
		t.Fatalf("expected 0 loss, got %f", lossSub)
	}
	if avgSub <= 0 {
		t.Fatalf("expected sub-ms avg > 0, got %f", avgSub)
	}
	if jitterSub < 0 {
		t.Fatalf("expected jitter >= 0, got %f", jitterSub)
	}

	// Partial loss Windows ping
	windowsPartial := `
Pinging 8.8.8.8 with 32 bytes of data:
Reply from 8.8.8.8: bytes=32 time=20ms TTL=117
Request timed out.
Reply from 8.8.8.8: bytes=32 time=30ms TTL=117
Reply from 8.8.8.8: bytes=32 time=25ms TTL=117

Ping statistics for 8.8.8.8:
    Packets: Sent = 4, Received = 3, Lost = 1 (25% loss),
`
	avgPart, jitterPart, lossPart, errPart := parsePingWindows(windowsPartial, 4)
	if errPart != nil {
		t.Fatalf("unexpected error on partial loss: %v", errPart)
	}
	if lossPart != 25.0 {
		t.Fatalf("expected 25%% loss, got %f", lossPart)
	}
	if avgPart != 25.0 { // (20+30+25)/3 = 25
		t.Fatalf("expected avg 25.0, got %f", avgPart)
	}
	if jitterPart != 7.5 { // (|30-20| + |25-30|)/2 = (10+5)/2 = 7.5
		t.Fatalf("expected jitter 7.5, got %f", jitterPart)
	}

	// 100% loss Windows ping
	windows100 := `
Pinging 10.255.255.1 with 32 bytes of data:
Request timed out.
Request timed out.
Request timed out.
Request timed out.

Ping statistics for 10.255.255.1:
    Packets: Sent = 4, Received = 0, Lost = 4 (100% loss),
`
	_, _, loss100, err100 := parsePingWindows(windows100, 4)
	if err100 == nil {
		t.Fatalf("expected error on 100%% loss, got nil")
	}
	if loss100 != 100.0 {
		t.Fatalf("expected 100%% loss, got %f", loss100)
	}

	// Host not found
	windowsHostNotFound := `Ping request could not find host nonexistinghost98765.com. Please check the name and try again.`
	_, _, lossHnf, errHnf := parsePingWindows(windowsHostNotFound, 4)
	if errHnf == nil {
		t.Fatalf("expected error on host not found, got nil")
	}
	if lossHnf != 100.0 {
		t.Fatalf("expected 100%% loss on host not found, got %f", lossHnf)
	}
}

func TestParsePingUnix(t *testing.T) {
	// Linux ping output
	linuxOutput := `
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=10.0 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=12.0 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=11.0 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=117 time=15.0 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 10.000/12.000/15.000/1.870 ms
`
	avg, jitter, loss, err := parsePingUnix(linuxOutput, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loss != 0 {
		t.Fatalf("expected 0 loss, got %f", loss)
	}
	if avg != 12.0 {
		t.Fatalf("expected avg 12.0, got %f", avg)
	}
	expectedJitter := 7.0 / 3.0
	if math.Abs(jitter-expectedJitter) > 1e-4 {
		t.Fatalf("expected jitter %f, got %f", expectedJitter, jitter)
	}

	// macOS ping output
	macOutput := `
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: icmp_seq=0 ttl=117 time=14.234 ms
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=16.892 ms

--- 8.8.8.8 ping statistics ---
2 packets transmitted, 2 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 14.234/15.563/16.892/1.329 ms
`
	avgMac, _, lossMac, errMac := parsePingUnix(macOutput, 2)
	if errMac != nil {
		t.Fatalf("unexpected error on mac: %v", errMac)
	}
	if lossMac != 0 {
		t.Fatalf("expected 0 loss, got %f", lossMac)
	}
	if avgMac < 15.0 || avgMac > 16.0 {
		t.Fatalf("unexpected avg %f", avgMac)
	}

	// Host not found
	unixNotFound := `ping: cannot resolve nonexistentdomain12345.com: Unknown host`
	_, _, lossUnf, errUnf := parsePingUnix(unixNotFound, 4)
	if errUnf == nil {
		t.Fatalf("expected error on unix unknown host, got nil")
	}
	if lossUnf != 100 {
		t.Fatalf("expected 100%% loss on unknown host, got %f", lossUnf)
	}
}

func TestLatencyAndLossScenarios(t *testing.T) {
	cases := []struct {
		name         string
		rawWindows   string
		count        int
		expectedAvg  float64
		expectedLoss float64
		expectErr    bool
	}{
		{
			name: "0% loss normal latency",
			rawWindows: `
Reply from 1.1.1.1: bytes=32 time=20ms TTL=57
Reply from 1.1.1.1: bytes=32 time=22ms TTL=57
Reply from 1.1.1.1: bytes=32 time=18ms TTL=57
Reply from 1.1.1.1: bytes=32 time=20ms TTL=57
`,
			count:        4,
			expectedAvg:  20.0,
			expectedLoss: 0.0,
			expectErr:    false,
		},
		{
			name: "High latency (500ms)",
			rawWindows: `
Reply from 1.1.1.1: bytes=32 time=500ms TTL=57
Reply from 1.1.1.1: bytes=32 time=520ms TTL=57
Reply from 1.1.1.1: bytes=32 time=480ms TTL=57
`,
			count:        3,
			expectedAvg:  500.0,
			expectedLoss: 0.0,
			expectErr:    false,
		},
		{
			name: "25% loss (1 timed out out of 4)",
			rawWindows: `
Reply from 1.1.1.1: bytes=32 time=15ms TTL=57
Request timed out.
Reply from 1.1.1.1: bytes=32 time=15ms TTL=57
Reply from 1.1.1.1: bytes=32 time=15ms TTL=57
`,
			count:        4,
			expectedAvg:  15.0,
			expectedLoss: 25.0,
			expectErr:    false,
		},
		{
			name: "50% loss (2 timed out out of 4)",
			rawWindows: `
Request timed out.
Reply from 1.1.1.1: bytes=32 time=30ms TTL=57
Request timed out.
Reply from 1.1.1.1: bytes=32 time=30ms TTL=57
`,
			count:        4,
			expectedAvg:  30.0,
			expectedLoss: 50.0,
			expectErr:    false,
		},
		{
			name: "100% loss (all timed out)",
			rawWindows: `
Request timed out.
Request timed out.
Request timed out.
Request timed out.
`,
			count:        4,
			expectedAvg:  0.0,
			expectedLoss: 100.0,
			expectErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			avg, _, loss, err := parsePingWindows(tc.rawWindows, tc.count)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(loss-tc.expectedLoss) > 1e-2 {
				t.Fatalf("expected loss %f, got %f", tc.expectedLoss, loss)
			}
			if !tc.expectErr && math.Abs(avg-tc.expectedAvg) > 1e-2 {
				t.Fatalf("expected avg %f, got %f", tc.expectedAvg, avg)
			}
		})
	}
}

func startMockDNSServer(t *testing.T, handler func(query []byte) ([]byte, time.Duration)) (string, func()) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("failed to start mock DNS UDP server: %v", err)
	}

	stopChan := make(chan struct{})
	go func() {
		buf := make([]byte, 512)
		for {
			n, remoteAddr, err := conn.ReadFrom(buf)
			if err != nil {
				select {
				case <-stopChan:
					return
				default:
					return
				}
			}
			if handler != nil {
				queryCopy := make([]byte, n)
				copy(queryCopy, buf[:n])
				resp, delay := handler(queryCopy)
				if delay > 0 {
					time.Sleep(delay)
				}
				if resp != nil {
					_, _ = conn.WriteTo(resp, remoteAddr)
				}
			}
		}
	}()

	return conn.LocalAddr().String(), func() {
		close(stopChan)
		_ = conn.Close()
	}
}

func TestMeasureDNSMockServer(t *testing.T) {
	// 1. Successful DNS response
	t.Run("Successful Resolution", func(t *testing.T) {
		addr, closeFn := startMockDNSServer(t, func(query []byte) ([]byte, time.Duration) {
			if len(query) < 12 {
				return nil, 0
			}
			// Copy query and create standard 12-byte response header with QR=1, RCODE=0
			resp := make([]byte, len(query))
			copy(resp, query)
			resp[2] = 0x81 // QR=1, RD=1, RA=1
			resp[3] = 0x80 // RCODE=0 (No error)
			return resp, 10 * time.Millisecond
		})
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		dur, err := measureDNS(ctx, "example.com", addr)
		if err != nil {
			t.Fatalf("expected successful resolution, got error: %v", err)
		}
		if dur <= 0 {
			t.Fatalf("expected positive duration, got %v", dur)
		}
	})

	// 2. Failed Resolution (RCODE 3 = NXDOMAIN)
	t.Run("Failed Resolution (NXDOMAIN)", func(t *testing.T) {
		addr, closeFn := startMockDNSServer(t, func(query []byte) ([]byte, time.Duration) {
			if len(query) < 12 {
				return nil, 0
			}
			resp := make([]byte, len(query))
			copy(resp, query)
			resp[2] = 0x81
			resp[3] = 0x83 // RCODE=3 (NXDOMAIN)
			return resp, 0
		})
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := measureDNS(ctx, "nonexistent.example.com", addr)
		if err == nil {
			t.Fatalf("expected error on RCODE=3, got nil")
		}
	})

	// 3. DNS Timeout / Slow server
	t.Run("DNS Timeout Handling", func(t *testing.T) {
		addr, closeFn := startMockDNSServer(t, func(query []byte) ([]byte, time.Duration) {
			// Don't respond, simulate dropped UDP packet
			return nil, 0
		})
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := measureDNS(ctx, "example.com", addr)
		if err == nil {
			t.Fatalf("expected timeout error, got nil")
		}
	})

	// 4. Transaction ID Mismatch
	t.Run("Transaction ID Mismatch", func(t *testing.T) {
		addr, closeFn := startMockDNSServer(t, func(query []byte) ([]byte, time.Duration) {
			resp := make([]byte, len(query))
			copy(resp, query)
			resp[0] ^= 0xFF // Corrupt transaction ID
			resp[2] = 0x81
			resp[3] = 0x80
			return resp, 0
		})
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := measureDNS(ctx, "example.com", addr)
		if err == nil {
			t.Fatalf("expected error on ID mismatch, got nil")
		}
	})

	// 5. Truncated Response (<12 bytes)
	t.Run("Truncated Response", func(t *testing.T) {
		addr, closeFn := startMockDNSServer(t, func(query []byte) ([]byte, time.Duration) {
			return []byte{0x01, 0x02}, 0 // Only 2 bytes
		})
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := measureDNS(ctx, "example.com", addr)
		if err == nil {
			t.Fatalf("expected error on truncated response, got nil")
		}
	})
}
