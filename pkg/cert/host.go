package cert

import (
	"net"
	"os"
)

// DefaultSANs returns localhost, loopback, hostname, and primary local IPv4.
func DefaultSANs() []string {
	out := []string{"localhost", "127.0.0.1", "::1"}
	if h, err := os.Hostname(); err == nil && h != "" {
		out = append(out, h)
	}
	if ip := detectLocalIP(); ip != "" {
		out = append(out, ip)
	}
	return out
}

func detectLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		return ipNet.IP.String()
	}
	return ""
}
