package network

import (
	"math"
	"testing"
	"time"
)

func TestCalculateThroughput(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(2 * time.Second) // 2.0 seconds elapsed

	// 5,000,000 bytes Rx in 2 seconds = 2,500,000 B/s = 20.0 Mbps
	// 1,000,000 bytes Tx in 2 seconds = 500,000 B/s = 4.0 Mbps
	prev := InterfaceStats{
		Name:      "Wi-Fi",
		RxBytes:   10000000,
		TxBytes:   20000000,
		Timestamp: t1,
	}
	curr := InterfaceStats{
		Name:      "Wi-Fi",
		RxBytes:   15000000,
		TxBytes:   21000000,
		Timestamp: t2,
	}

	rxMbps, txMbps, rxBytesSec, txBytesSec := CalculateThroughput(prev, curr)

	if math.Abs(rxMbps-20.0) > 1e-2 {
		t.Fatalf("expected rxMbps 20.0, got %f", rxMbps)
	}
	if math.Abs(txMbps-4.0) > 1e-2 {
		t.Fatalf("expected txMbps 4.0, got %f", txMbps)
	}
	if math.Abs(rxBytesSec-2500000.0) > 1e-2 {
		t.Fatalf("expected rxBytesSec 2500000.0, got %f", rxBytesSec)
	}
	if math.Abs(txBytesSec-500000.0) > 1e-2 {
		t.Fatalf("expected txBytesSec 500000.0, got %f", txBytesSec)
	}

	// Test Counter Reset / Wrap-around (curr < prev)
	currReset := InterfaceStats{
		Name:      "Wi-Fi",
		RxBytes:   500,
		TxBytes:   200,
		Timestamp: t2,
	}
	rxMReset, txMReset, _, _ := CalculateThroughput(prev, currReset)
	if rxMReset != 0 || txMReset != 0 {
		t.Fatalf("expected 0 rate on counter reset, got rx=%f tx=%f", rxMReset, txMReset)
	}

	// Test Unchanged Counters (Idle network)
	currUnchanged := InterfaceStats{
		Name:      "Wi-Fi",
		RxBytes:   10000000,
		TxBytes:   20000000,
		Timestamp: t2,
	}
	rxIdle, txIdle, rxBpsIdle, txBpsIdle := CalculateThroughput(prev, currUnchanged)
	if rxIdle != 0 || txIdle != 0 || rxBpsIdle != 0 || txBpsIdle != 0 {
		t.Fatalf("expected 0 rate for unchanged counters, got rx=%f, tx=%f", rxIdle, txIdle)
	}

	// Test Zero Duration
	rxZero, _, _, _ := CalculateThroughput(prev, prev)
	if rxZero != 0 {
		t.Fatalf("expected 0 for zero duration")
	}

	// Test Negative Duration (curr before prev)
	rxNeg, _, _, _ := CalculateThroughput(curr, prev)
	if rxNeg != 0 {
		t.Fatalf("expected 0 for negative duration")
	}
}

func TestParseWindowsSubinterfaces(t *testing.T) {
	sampleOutput := `
       MTU  MediaSenseState      Bytes In     Bytes Out  Interface
----------  ---------------  ------------  ------------  -------------
4294967295                1             0         19385  Loopback Pseudo-Interface 1
      1500                1    2344979525    4065941533  Wi-Fi
      1500                5             0             0  Local Area Connection* 1
      1500                5             0             0  Bluetooth Network Connection
      1500                5             0             0  Local Area Connection* 2
`
	stats, err := parseWindowsSubinterfaces(sampleOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Name != "Wi-Fi" {
		t.Fatalf("expected interface 'Wi-Fi', got '%s'", stats.Name)
	}
	if stats.RxBytes != 2344979525 {
		t.Fatalf("expected RxBytes 2344979525, got %d", stats.RxBytes)
	}
	if stats.TxBytes != 4065941533 {
		t.Fatalf("expected TxBytes 4065941533, got %d", stats.TxBytes)
	}
}

func TestParseLinuxProcNetDev(t *testing.T) {
	sampleDev := []byte(`Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 120485760   17500    0    0    0     0          0         0 120485760   17500    0    0    0     0       0          0
  eth0: 523456789   45000    0    0    0     0          0         0 123456789   35000    0    0    0     0       0          0
docker0:        0       0    0    0    0     0          0         0         0       0    0    0    0     0       0          0
`)
	stats, err := parseLinuxProcNetDev(sampleDev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Name != "eth0" {
		t.Fatalf("expected interface 'eth0', got '%s'", stats.Name)
	}
	if stats.RxBytes != 523456789 {
		t.Fatalf("expected RxBytes 523456789, got %d", stats.RxBytes)
	}
	if stats.TxBytes != 123456789 {
		t.Fatalf("expected TxBytes 123456789, got %d", stats.TxBytes)
	}
}
