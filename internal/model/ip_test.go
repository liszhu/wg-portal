package model

/*
func ignoreNetlinkError(addr *netlink.Addr, _ error) *netlink.Addr {
	return addr
}

func TestBroadcastAddr(t *testing.T) {
	tests := []struct {
		name string
		arg  *netlink.Addr
		want *netlink.Addr
	}{
		{
			name: "V4_0",
			arg:  ignoreNetlinkError(common.ParseCidr("10.0.0.0/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
		},
		{
			name: "V4_1",
			arg:  ignoreNetlinkError(common.ParseCidr("10.0.0.1/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
		},
		{
			name: "V4_2",
			arg:  ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
		},
		{
			name: "V6_0",
			arg:  ignoreNetlinkError(common.ParseCidr("fe80::/64")),
			want: ignoreNetlinkError(common.ParseCidr("fe80::ffff:ffff:ffff:ffff/64")),
		},
		{
			name: "V6_1",
			arg:  ignoreNetlinkError(common.ParseCidr("fe80::1:2:3/64")),
			want: ignoreNetlinkError(common.ParseCidr("fe80::ffff:ffff:ffff:ffff/64")),
		},
		{
			name: "V6_2",
			arg:  ignoreNetlinkError(common.ParseCidr("fe80::ffff:ffff:ffff:ffff/64")),
			want: ignoreNetlinkError(common.ParseCidr("fe80::ffff:ffff:ffff:ffff/64")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.BroadcastAddr(tt.arg); got.String() != tt.want.String() {
				t.Errorf("broadcastAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpAddressesFromString(t *testing.T) {
	type args struct {
		addrStr string
	}
	var tests = []struct {
		name    string
		args    args
		want    []*netlink.Addr
		wantErr bool
	}{
		{
			name:    "Empty String",
			args:    args{},
			want:    []*netlink.Addr{},
			wantErr: false,
		},
		{
			name:    "Single IPv4",
			args:    args{addrStr: "123.123.123.123"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Malformed",
			args:    args{addrStr: "hello world"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single IPv4 CIDR",
			args: args{addrStr: "123.123.123.123/24"},
			want: []*netlink.Addr{{
				IPNet: &net.IPNet{
					IP:   net.IPv4(123, 123, 123, 123),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			}},
			wantErr: false,
		},
		{
			name: "Multiple IPv4 CIDR",
			args: args{addrStr: "123.123.123.123/24, 200.201.202.203/16"},
			want: []*netlink.Addr{{
				IPNet: &net.IPNet{
					IP:   net.IPv4(123, 123, 123, 123),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			}, {
				IPNet: &net.IPNet{
					IP:   net.IPv4(200, 201, 202, 203),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}},
			wantErr: false,
		},
		{
			name: "Single IPv6 CIDR",
			args: args{addrStr: "fe80::1/64"},
			want: []*netlink.Addr{{
				IPNet: &net.IPNet{
					IP:   net.IP{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01},
					Mask: net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0},
				},
			}},
			wantErr: false,
		},
		{
			name: "Multiple IPv6 CIDR",
			args: args{addrStr: "fe80::1/64 , 2130:d3ad::b33f/128"},
			want: []*netlink.Addr{{
				IPNet: &net.IPNet{
					IP:   net.IP{0x21, 0x30, 0xd3, 0xad, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xb3, 0x3f},
					Mask: net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				},
			}, {
				IPNet: &net.IPNet{
					IP:   net.IP{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01},
					Mask: net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0},
				},
			}},
			wantErr: false,
		},
		{
			name: "Mixed IPv4 and IPv6 CIDR",
			args: args{addrStr: "200.201.202.203/16,2130:d3ad::b33f/128"},
			want: []*netlink.Addr{{
				IPNet: &net.IPNet{
					IP:   net.IPv4(200, 201, 202, 203),
					Mask: net.IPv4Mask(255, 255, 0, 0),
				},
			}, {
				IPNet: &net.IPNet{
					IP:   net.IP{0x21, 0x30, 0xd3, 0xad, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xb3, 0x3f},
					Mask: net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.IpAddressesFromString(tt.args.addrStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIpAddressString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIpAddressString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpAddressesToString_netlink(t *testing.T) {
	tests := []struct {
		name      string
		addresses []netlink.Addr
		want      string
	}{
		{
			name:      "Single",
			addresses: []netlink.Addr{*ignoreNetlinkError(common.ParseCidr("10.0.0.2/24"))},
			want:      "10.0.0.2/24",
		},
		{
			name: "Multiple",
			addresses: []netlink.Addr{
				*ignoreNetlinkError(common.ParseCidr("10.0.0.3/24")),
				*ignoreNetlinkError(common.ParseCidr("10.0.0.2/24")),
				*ignoreNetlinkError(common.ParseCidr("8.8.8.8/32")),
				*ignoreNetlinkError(common.ParseCidr("fe80::/64")),
			},
			want: "10.0.0.3/24,10.0.0.2/24,8.8.8.8/32,fe80::/64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.IpAddressesToString(tt.addresses); got != tt.want {
				t.Errorf("ipAddressesToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpAddressesToString_net(t *testing.T) {
	tests := []struct {
		name      string
		addresses []net.IPNet
		want      string
	}{
		{
			name: "Single",
			addresses: []net.IPNet{{
				IP:   net.IPv4(10, 0, 0, 2),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			}},
			want: "10.0.0.2/24",
		},
		{
			name: "Multiple",
			addresses: []net.IPNet{{
				IP:   net.IPv4(10, 0, 0, 3),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			}, {
				IP:   net.IPv4(10, 0, 0, 2),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			}, {
				IP:   net.IPv6loopback,
				Mask: net.CIDRMask(64, 128),
			}},
			want: "10.0.0.3/24,10.0.0.2/24,::1/64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.IpAddressesToString(tt.addresses); got != tt.want {
				t.Errorf("ipAddressesToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpIncrease(t *testing.T) {
	tests := []struct {
		name string
		ip   *netlink.Addr
		want *netlink.Addr
	}{
		{
			name: "V4_1",
			ip:   ignoreNetlinkError(common.ParseCidr("10.0.0.0/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.1/24")),
		},
		{
			name: "V4_2",
			ip:   ignoreNetlinkError(common.ParseCidr("10.0.0.2/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.3/24")),
		},
		{
			name: "V4_3",
			ip:   ignoreNetlinkError(common.ParseCidr("10.0.0.254/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
		},
		{
			name: "V4_4",
			ip:   ignoreNetlinkError(common.ParseCidr("10.0.0.255/24")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.1.0/24")),
		},
		{
			name: "V4_5",
			ip:   ignoreNetlinkError(common.ParseCidr("10.0.0.5/32")),
			want: ignoreNetlinkError(common.ParseCidr("10.0.0.6/32")),
		},
		{
			name: "V6_1",
			ip:   ignoreNetlinkError(common.ParseCidr("2001:db8::/64")),
			want: ignoreNetlinkError(common.ParseCidr("2001:db8::1/64")),
		},
		{
			name: "V6_2",
			ip:   ignoreNetlinkError(common.ParseCidr("2001:db8::5/64")),
			want: ignoreNetlinkError(common.ParseCidr("2001:db8::6/64")),
		},
		{
			name: "V6_3",
			ip:   ignoreNetlinkError(common.ParseCidr("2001:0db8:0000:0000:ffff:ffff:ffff:fffe/64")),
			want: ignoreNetlinkError(common.ParseCidr("2001:0db8:0000:0000:ffff:ffff:ffff:ffff/64")),
		},
		{
			name: "V6_4",
			ip:   ignoreNetlinkError(common.ParseCidr("2001:0db8:0:0:ffff:ffff:ffff:ffff/64")),
			want: ignoreNetlinkError(common.ParseCidr("2001:db8:0:1::/64")),
		},
		{
			name: "V6_5",
			ip:   ignoreNetlinkError(common.ParseCidr("2001:0db8:0:0:ffff:ffff:ffff:ffff/128")),
			want: ignoreNetlinkError(common.ParseCidr("2001:0db8:0:1::/128")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.IpIncrease(tt.ip)
			if !reflect.DeepEqual(tt.ip, tt.want) {
				t.Errorf("increaseIP() got = %v, want %v", tt.ip, tt.want)
			}
		})
	}
}

func TestIpIsV4(t *testing.T) {
	tests := []struct {
		name string
		arg  *netlink.Addr
		want bool
	}{
		{
			name: "V4",
			arg:  ignoreNetlinkError(common.ParseCidr("10.0.0.1/24")),
			want: true,
		},
		{
			name: "V4 network",
			arg:  ignoreNetlinkError(common.ParseCidr("10.0.0.0/24")),
			want: true,
		},
		{
			name: "V6",
			arg:  ignoreNetlinkError(common.ParseCidr("fe80::/64")),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.IpIsV4(tt.arg); got != tt.want {
				t.Errorf("isV4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		want    *netlink.Addr
		wantErr bool
	}{
		{
			name: "Valid V4",
			cidr: "10.0.0.1/24",
			want: &netlink.Addr{IPNet: &net.IPNet{
				IP:   net.IPv4(10, 0, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			wantErr: false,
		},
		{
			name:    "Inalid V4",
			cidr:    "10.0.0.1/64",
			want:    nil,
			wantErr: true,
		},
		{
			name: "Valid V6",
			cidr: "fe80::/128",
			want: &netlink.Addr{IPNet: &net.IPNet{
				IP:   []byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Mask: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			},
			wantErr: false,
		},
		{
			name:    "Inalid V6",
			cidr:    "10:0:0::/256",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.ParseCidr(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCidr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCidr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
