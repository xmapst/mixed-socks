package auth

import (
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/service"
	"net"
)

type Authenticator interface {
	Verify(user string, pass string, host string) bool
	Enable() bool
}

type User struct {
}

func (u *User) Verify(user, pass, host string) bool {
	res := service.GetUser(user)
	if res == nil {
		return false
	}
	if pass != "" {
		if res.Password != pass {
			return false
		}
	}
	src := net.ParseIP(host)
	for _, ipMask := range res.CIDR {
		if ip := net.ParseIP(ipMask); ip != nil {
			if ip.Equal(src) {
				return true
			}
		} else if _, ipNet, err := net.ParseCIDR(ipMask); err != nil {
			logrus.Errorln(err)
			continue
		} else {
			if ipNet.Contains(src) {
				return true
			}
		}
	}
	return true
}

func (u *User) Enable() bool {
	res, _ := service.ListUser()
	if res == nil {
		return false
	}
	return true
}
