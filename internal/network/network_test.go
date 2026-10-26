package network

import (
	"testing"
	"time"
)

func TestDetectAndGateway(t *testing.T) {
	// Should never panic regardless of platform or interface state
	info := Detect()
	if info.ID == "" {
		t.Fatalf("expected non-empty network ID")
	}

	gw := DefaultGateway()
	// gw may be empty if offline, but must not panic
	_ = gw
}

func TestExecCommandWithTimeout(t *testing.T) {
	// 1. Successful quick command (e.g. echo or whoami or ping -h or invalid flag)
	out, err := ExecCommandWithTimeout(1*time.Second, "cmd", "/c", "echo netmon_test")
	if err == nil {
		if len(out) == 0 {
			t.Fatalf("expected output from echo")
		}
	}

	// 2. Command with very short timeout should terminate cleanly without hanging
	start := time.Now()
	_, _ = ExecCommandWithTimeout(50*time.Millisecond, "ping", "-n", "10", "127.0.0.1")
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		t.Fatalf("command took %v, timeout was not respected", elapsed)
	}
}
