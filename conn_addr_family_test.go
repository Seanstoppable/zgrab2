package zgrab2

import (
	"net"
	"testing"
)

func TestFilterByAddressFamily(t *testing.T) {
	ipv4a := net.ParseIP("10.0.0.1")
	ipv4b := net.ParseIP("10.0.0.2")
	ipv6a := net.ParseIP("2001:db8::1")
	ipv6b := net.ParseIP("2001:db8::2")

	tests := []struct {
		name     string
		localIPs []net.IP
		targetIP net.IP
		wantLen  int
		wantIPv4 bool // whether results should all be IPv4
	}{
		{
			name:     "ipv4 target with mixed pool returns only ipv4",
			localIPs: []net.IP{ipv4a, ipv4b, ipv6a, ipv6b},
			targetIP: net.ParseIP("192.168.1.1"),
			wantLen:  2,
			wantIPv4: true,
		},
		{
			name:     "ipv6 target with mixed pool returns only ipv6",
			localIPs: []net.IP{ipv4a, ipv4b, ipv6a, ipv6b},
			targetIP: net.ParseIP("2607:f8b0:400a:809::200e"),
			wantLen:  2,
			wantIPv4: false,
		},
		{
			name:     "ipv4 target with only ipv4 pool returns all",
			localIPs: []net.IP{ipv4a, ipv4b},
			targetIP: net.ParseIP("192.168.1.1"),
			wantLen:  2,
			wantIPv4: true,
		},
		{
			name:     "ipv6 target with only ipv4 pool returns empty",
			localIPs: []net.IP{ipv4a, ipv4b},
			targetIP: net.ParseIP("2607:f8b0:400a:809::200e"),
			wantLen:  0,
		},
		{
			name:     "ipv4 target with only ipv6 pool returns empty",
			localIPs: []net.IP{ipv6a, ipv6b},
			targetIP: net.ParseIP("192.168.1.1"),
			wantLen:  0,
		},
		{
			name:     "nil target returns all local IPs unfiltered",
			localIPs: []net.IP{ipv4a, ipv6a},
			targetIP: nil,
			wantLen:  2,
		},
		{
			name:     "empty local IPs returns empty",
			localIPs: []net.IP{},
			targetIP: net.ParseIP("192.168.1.1"),
			wantLen:  0,
		},
		{
			name:     "nil local IPs returns nil",
			localIPs: nil,
			targetIP: net.ParseIP("192.168.1.1"),
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterByAddressFamily(tt.localIPs, tt.targetIP)
			if len(result) != tt.wantLen {
				t.Errorf("filterByAddressFamily() returned %d IPs, want %d", len(result), tt.wantLen)
			}
			if tt.wantLen > 0 && tt.targetIP != nil {
				for _, ip := range result {
					isIPv4 := ip.To4() != nil
					if isIPv4 != tt.wantIPv4 {
						t.Errorf("filterByAddressFamily() returned IP %s with wrong family", ip)
					}
				}
			}
		})
	}
}

func TestSetRandomLocalAddrFamilyMismatchError(t *testing.T) {
	d := NewDialer(nil)
	ipv4Only := []net.IP{net.ParseIP("10.0.0.1")}
	ipv6Target := net.ParseIP("2001:db8::1")

	err := d.SetRandomLocalAddr("tcp", ipv4Only, nil, ipv6Target)
	if err == nil {
		t.Fatal("SetRandomLocalAddr() should error when no local IPs match target family")
	}
}

func TestSetRandomLocalAddrFamilyMatch(t *testing.T) {
	d := NewDialer(nil)
	mixed := []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("2001:db8::1")}

	// IPv4 target should get an IPv4 local address
	err := d.SetRandomLocalAddr("tcp", mixed, nil, net.ParseIP("192.168.1.1"))
	if err != nil {
		t.Fatalf("SetRandomLocalAddr() unexpected error: %v", err)
	}
	tcpAddr, ok := d.LocalAddr.(*net.TCPAddr)
	if !ok {
		t.Fatal("LocalAddr should be *net.TCPAddr")
	}
	if tcpAddr.IP.To4() == nil {
		t.Errorf("Expected IPv4 local address, got %s", tcpAddr.IP)
	}

	// IPv6 target should get an IPv6 local address
	d2 := NewDialer(nil)
	err = d2.SetRandomLocalAddr("tcp", mixed, nil, net.ParseIP("2607:f8b0::1"))
	if err != nil {
		t.Fatalf("SetRandomLocalAddr() unexpected error: %v", err)
	}
	tcpAddr2, ok := d2.LocalAddr.(*net.TCPAddr)
	if !ok {
		t.Fatal("LocalAddr should be *net.TCPAddr")
	}
	if tcpAddr2.IP.To4() != nil {
		t.Errorf("Expected IPv6 local address, got %s", tcpAddr2.IP)
	}
}

func TestSetRandomLocalAddrNilTargetDefersBinding(t *testing.T) {
	d := NewDialer(nil)
	mixed := []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("2001:db8::1")}

	// nil target with local IPs should NOT set any local address (defers to DialContext)
	err := d.SetRandomLocalAddr("tcp", mixed, nil, nil)
	if err != nil {
		t.Fatalf("SetRandomLocalAddr() unexpected error: %v", err)
	}
	if d.LocalAddr != nil {
		t.Errorf("Expected nil LocalAddr when targetIP is nil, got %v", d.LocalAddr)
	}
}
