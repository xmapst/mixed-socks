package auth

import (
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/service"
	"net"
)

func VerifyIP(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	cidr := service.GetConf().CIDR
	if len(cidr) == 0 {
		return true
	}
	src := net.ParseIP(host)
	for _, ipMask := range cidr {
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
	return false
}
