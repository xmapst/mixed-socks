package auth

import (
	"github/xmapst/mixed-socks/internal/service"
)

type Authenticator interface {
	Verify(user string, pass string, host string) bool
	Enable() bool
}

type Auth struct {
}

func (a *Auth) Verify(u, p, h string) bool {
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
	return verifyCIDR(h, res.CIDR)
}

func (a *Auth) Enable() bool {
	auth := &service.Auth{}
	return auth.Get()
}
