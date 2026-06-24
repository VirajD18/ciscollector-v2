package cert

import (
	"net"
	"testing"
)

func TestParseSANList(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		wantDNS int
		wantIP  int
		wantErr bool
	}{
		{name: "dns_and_ip", in: []string{"localhost", "127.0.0.1", "192.168.1.50"}, wantDNS: 1, wantIP: 2},
		{name: "dedupe", in: []string{"localhost", "localhost", "10.0.0.1"}, wantDNS: 1, wantIP: 1},
		{name: "empty", in: []string{}, wantErr: true},
		{name: "ipv6", in: []string{"::1"}, wantIP: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSANList(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			dns, ips := sanEntriesToX509(got)
			if len(dns) != tc.wantDNS {
				t.Fatalf("dns=%d want %d", len(dns), tc.wantDNS)
			}
			if len(ips) != tc.wantIP {
				t.Fatalf("ips=%d want %d", len(ips), tc.wantIP)
			}
		})
	}
}

func TestSplitSANArg(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "localhost,127.0.0.1,192.168.1.50", want: []string{"localhost", "127.0.0.1", "192.168.1.50"}},
		{in: "", want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := SplitSANArg(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestSANEntriesToX509(t *testing.T) {
	entries := []sanEntry{{dns: "host"}, {ip: net.ParseIP("10.0.0.5")}}
	dns, ips := sanEntriesToX509(entries)
	if len(dns) != 1 || dns[0] != "host" {
		t.Fatalf("dns=%v", dns)
	}
	if len(ips) != 1 || ips[0].String() != "10.0.0.5" {
		t.Fatalf("ips=%v", ips)
	}
}
