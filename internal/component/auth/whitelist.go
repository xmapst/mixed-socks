package auth

import (
	"net"
)

type Whitelist interface {
	Verify(addr string) bool
	IPs() []net.IP
}

type inMemoryWhitelist struct {
	IP []net.IP
}

func (au *inMemoryWhitelist) Verify(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	src := net.ParseIP(host)
	for _, ip := range au.IP {
		if ip.Equal(src) {
			return true
		}
	}
	return false
}

func (au *inMemoryWhitelist) IPs() []net.IP { return au.IP }

func NewWhitelist(ip []net.IP) Whitelist {
	if len(ip) == 0 {
		return nil
	}

	au := &inMemoryWhitelist{
		IP: ip,
	}
	return au
}
