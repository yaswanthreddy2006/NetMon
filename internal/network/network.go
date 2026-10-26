package network

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DefaultCommandTimeout is the maximum time allowed for local network system commands.
const DefaultCommandTimeout = 3 * time.Second

// ExecCommandWithTimeout runs an external command with a strict timeout context to prevent hangs.
func ExecCommandWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// ExecCommandCombinedWithTimeout runs an external command with a strict timeout context returning combined output.
func ExecCommandCombinedWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Info describes the current network the device is on.
type Info struct {
	// ID is a stable identifier: SSID name for Wi-Fi, or "gw:<ip>" for wired/Docker/unknown.
	ID string
	// SSID is the Wi-Fi network name, empty if wired or not detectable.
	SSID string
	// Gateway is the default gateway IP.
	Gateway string
	// LocalIP is the outbound local IPv4 address of the active network interface.
	LocalIP string
}

// OutboundIP returns the local IP address used for outbound communication.
func OutboundIP() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 1*time.Second)
	if err != nil {
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
					continue
				}
				addrs, err := iface.Addrs()
				if err != nil {
					continue
				}
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
						return ipNet.IP.String()
					}
				}
			}
		}
		return ""
	}
	defer conn.Close()
	if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return localAddr.IP.String()
	}
	return ""
}

// Detect returns the current network identity using SSID when available,
// falling back to the default gateway IP.
// Works on Linux (including Raspberry Pi and Docker), macOS, and Windows.
func Detect() Info {
	gw := DefaultGateway()
	ssid := detectSSID()
	localIP := OutboundIP()

	id := "unknown"
	switch {
	case ssid != "":
		id = ssid
	case gw != "":
		id = "gw:" + gw
	}

	return Info{ID: id, SSID: ssid, Gateway: gw, LocalIP: localIP}
}

// DefaultGateway returns the default gateway IP in a cross-platform way.
func DefaultGateway() string {
	switch runtime.GOOS {
	case "windows":
		return gatewayWindows()
	case "linux":
		if gw := gatewayLinuxProc(); gw != "" {
			return gw
		}
		return gatewayLinuxCmd()
	default: // darwin, freebsd, openbsd
		return gatewayDarwin()
	}
}

// darwinRoute parses "route get default" and returns the default (gateway, interface).
// Shared by DefaultGateway and GetConnectionInfo to avoid running the command twice.
func darwinRoute() (gateway, iface string) {
	out, _ := ExecCommandCombinedWithTimeout(DefaultCommandTimeout, "route", "get", "default")
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "gateway:"):
			gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		case strings.HasPrefix(line, "interface:"):
			iface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	return
}

func gatewayDarwin() string {
	gw, _ := darwinRoute()
	return gw
}

// gatewayLinuxProc parses /proc/net/route for the default route (no external commands).
func gatewayLinuxProc() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}
		// Gateway field is 8-hex-char little-endian: "0101A8C0" → 192.168.1.1
		b, err := hex.DecodeString(fields[2])
		if err != nil || len(b) < 4 {
			continue
		}
		return fmt.Sprintf("%d.%d.%d.%d", b[3], b[2], b[1], b[0])
	}
	return ""
}

func gatewayLinuxCmd() string {
	out, err := ExecCommandCombinedWithTimeout(DefaultCommandTimeout, "ip", "route", "show", "default")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "default via ") {
			if parts := strings.Fields(line); len(parts) >= 3 {
				if net.ParseIP(parts[2]) != nil {
					return parts[2]
				}
			}
		}
	}
	return ""
}

func gatewayWindows() string {
	out, err := ExecCommandCombinedWithTimeout(DefaultCommandTimeout, "route", "print", "0.0.0.0")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			if net.ParseIP(fields[2]) != nil {
				return fields[2]
			}
		}
	}
	return ""
}

// detectSSID returns the Wi-Fi SSID of the current connection,
// or "" if not on Wi-Fi or not detectable (e.g. wired, Docker).
func detectSSID() string {
	switch runtime.GOOS {
	case "darwin":
		return ssidDarwin()
	case "linux":
		return ssidLinux()
	case "windows":
		return ssidWindows()
	}
	return ""
}

func ssidDarwin() string {
	for _, iface := range []string{"en0", "en1", "en2", "en3"} {
		out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "networksetup", "-getairportnetwork", iface)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(out))
		const prefix = "Current Wi-Fi Network: "
		if strings.HasPrefix(text, prefix) {
			return strings.TrimPrefix(text, prefix)
		}
	}
	return ""
}

func ssidLinux() string {
	if out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "iwgetid", "-r"); err == nil {
		if ssid := strings.TrimSpace(string(out)); ssid != "" {
			return ssid
		}
	}
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "iw", "dev")
	if err != nil {
		return ""
	}
	var iface string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Interface ") {
			iface = strings.TrimPrefix(line, "Interface ")
		} else if iface != "" && strings.HasPrefix(line, "ssid ") {
			return strings.TrimPrefix(line, "ssid ")
		}
	}
	return ""
}

func ssidWindows() string {
	out, err := ExecCommandWithTimeout(DefaultCommandTimeout, "netsh", "wlan", "show", "interfaces")
	if err != nil {
		return ""
	}
	var inConnectedBlock bool
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.TrimSpace(parts[1])
			if k == "state" || k == "status" {
				inConnectedBlock = (v == "connected" || strings.Contains(v, "verbunden"))
			}
			if inConnectedBlock && k == "ssid" {
				if v != "" {
					return v
				}
			}
		}
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "SSID" {
			val := strings.TrimSpace(parts[1])
			if val != "" {
				return val
			}
		}
	}
	return ""
}
