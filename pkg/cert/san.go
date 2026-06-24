package cert

import (
	"fmt"
	"net"
	"strings"
)

type sanEntry struct {
	dns string
	ip  net.IP
}

func parseSANList(raw []string) ([]sanEntry, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("at least one SAN is required")
	}
	out := make([]sanEntry, 0, len(raw))
	seen := make(map[string]struct{})
	for _, item := range raw {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		if ip := net.ParseIP(v); ip != nil {
			out = append(out, sanEntry{ip: ip})
			continue
		}
		out = append(out, sanEntry{dns: v})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one SAN is required")
	}
	return out, nil
}

// SplitSANArg parses a comma-separated SAN flag value.
func SplitSANArg(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func sanEntriesToX509(entries []sanEntry) (dns []string, ips []net.IP) {
	for _, e := range entries {
		if e.dns != "" {
			dns = append(dns, e.dns)
		}
		if e.ip != nil {
			ips = append(ips, e.ip)
		}
	}
	return dns, ips
}
