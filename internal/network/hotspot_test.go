package network

import (
	"testing"
)

func TestCalculateHotspotClientThroughput(t *testing.T) {
	t.Run("Zero elapsed returns 0", func(t *testing.T) {
		rx, tx, tot := CalculateHotspotClientThroughput(1000, 1000, 0)
		if rx != 0 || tx != 0 || tot != 0 {
			t.Errorf("expected 0, got rx=%f, tx=%f, tot=%f", rx, tx, tot)
		}
	})

	t.Run("1MB in 1s produces 8.0 Mbps", func(t *testing.T) {
		// 1,000,000 bytes * 8 bits / 1s / 1e6 = 8.0 Mbps
		rx, tx, tot := CalculateHotspotClientThroughput(1000000, 500000, 1.0)
		if rx != 8.0 {
			t.Errorf("expected 8.0 Mbps rx, got %f", rx)
		}
		if tx != 4.0 {
			t.Errorf("expected 4.0 Mbps tx, got %f", tx)
		}
		if tot != 12.0 {
			t.Errorf("expected 12.0 Mbps total, got %f", tot)
		}
	})
}

func TestCleanHostname(t *testing.T) {
	cases := []struct {
		raw      string
		ip       string
		expected string
	}{
		{"Yaswanth-reddy.mshome.net", "192.168.137.197", "Yaswanth Reddy"},
		{"samsung-galaxy-s23.local", "192.168.137.50", "Samsung Galaxy S23"},
		{"DESKTOP_ASUS.lan", "192.168.137.20", "DESKTOP ASUS"},
		{"", "192.168.137.100", "Unknown Device"},
		{"   ", "192.168.137.101", "Unknown Device"},
	}

	for _, c := range cases {
		out := CleanHostname(c.raw, c.ip)
		if out != c.expected {
			t.Errorf("CleanHostname(%q, %q) = %q; expected %q", c.raw, c.ip, out, c.expected)
		}
	}
}

func TestInferDeviceType(t *testing.T) {
	cases := []struct {
		hostname string
		mac      string
		expType  string
		expIcon  string
	}{
		{"Redmi-Note-13", "", "phone", "📱"},
		{"iPhone-15-Pro", "", "phone", "📱"},
		{"Galaxy-A55", "", "phone", "📱"},
		{"ASUS-ROG-Laptop", "", "laptop", "💻"},
		{"MacBook-Air", "", "laptop", "💻"},
		{"LG-SmartTV-4K", "", "tv", "📺"},
		{"Gaming-Desktop-PC", "", "desktop", "🖥️"},
		{"Generic-Node", "", "generic", "🔌"},
	}

	for _, c := range cases {
		dType, dIcon := InferDeviceType(c.hostname, c.mac)
		if dType != c.expType || dIcon != c.expIcon {
			t.Errorf("InferDeviceType(%q) = (%q, %q); expected (%q, %q)", c.hostname, dType, dIcon, c.expType, c.expIcon)
		}
	}
}

func TestAggregateHotspotClients(t *testing.T) {
	clients := []HotspotClient{
		{IP: "192.168.137.10", Hostname: "Phone A", RxMbps: 10.0, TxMbps: 2.0, TotalMbps: 12.0},
		{IP: "192.168.137.20", Hostname: "Laptop B", RxMbps: 20.0, TxMbps: 4.0, TotalMbps: 24.0},
		{IP: "192.168.137.30", Hostname: "TV C", RxMbps: 0.0, TxMbps: 0.0, TotalMbps: 0.0},
	}

	res := AggregateHotspotClients(clients, 36.0)

	if len(res) != 3 {
		t.Fatalf("expected 3 clients, got %d", len(res))
	}

	// Should be sorted descending by TotalMbps: Laptop B (24), Phone A (12), TV C (0)
	if res[0].Hostname != "Laptop B" || res[1].Hostname != "Phone A" || res[2].Hostname != "TV C" {
		t.Errorf("unexpected sort order: %+v", res)
	}

	// Check percentages: 24/36 = 66.7%, 12/36 = 33.3%, 0%
	if res[0].Percent < 66.0 || res[0].Percent > 67.0 {
		t.Errorf("expected ~66.7%%, got %f", res[0].Percent)
	}
	if !res[0].IsActive {
		t.Errorf("Laptop B should be marked active")
	}
	if res[2].IsActive {
		t.Errorf("TV C with 0 traffic should be idle")
	}
}
