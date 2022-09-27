package auth

import (
	"github/xmapst/mixed-socks/internal/service"
	"net"
)

type Authenticator interface {
	Verify(user string, pass string, addr string) bool
	Enable() bool
}

type Auth struct {
}

func (a *Auth) Verify(u, p, addr string) bool {
	user := &service.User{
		Name: u,
	}
	res, err := user.Get()
	if res == nil || err != nil {
		return false
	}
	if res.Disabled {
		return false
	}
	if p != "" {
		if res.Pass != p {
			return false
		}
	}
	if res.CIDR == nil {
		return true
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return verifyCIDR(host, res.CIDR)
}

func (a *Auth) Enable() bool {
	auth := &service.Auth{}
	return auth.Get()
}
