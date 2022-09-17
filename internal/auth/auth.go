package auth

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type Service interface {
	Verify(...string) bool
	Enable() bool
}

type PasswordAuth struct {
	storage   *sync.Map
	usernames []string
}

type User struct {
	Username string `yaml:""`
	Password string `yaml:""`
}

func (pa *PasswordAuth) New(list []User) error {
	if pa.storage == nil {
		pa.storage = &sync.Map{}
	}
	for _, user := range list {
		logrus.Infoln("access user:", user.Username)
		pa.storage.Store(user.Username, user.Password)
	}
	usernames := make([]string, 0, len(list))
	pa.storage.Range(func(key, value any) bool {
		usernames = append(usernames, key.(string))
		return true
	})
	pa.usernames = usernames
	return nil
}

func (pa *PasswordAuth) Verify(args ...string) bool {
	if len(args) < 2 {
		return false
	}
	user := args[0]
	pass := args[1]
	realPass, ok := pa.storage.Load(user)
	if pass == "" {
		return ok
	}
	return ok && realPass == pass
}

func (pa *PasswordAuth) Enable() bool {
	if pa.usernames == nil {
		return false
	}
	return true
}

type IPAuth struct {
	ips    []*net.IP
	ipsNet []*net.IPNet
}

func (i *IPAuth) Verify(args ...string) bool {
	if len(args) < 1 {
		return false
	}
	addr := args[0]
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return i.contains(host)
}

func (i *IPAuth) Enable() bool {
	if i.ips == nil && i.ipsNet == nil {
		return false
	}
	return true
}

func (i *IPAuth) New(list []string) error {
	if len(list) == 0 {
		return nil
	}
	list = append(list, "127.0.0.1/32")
	var (
		ips    []*net.IP
		ipsNet []*net.IPNet
	)

	for _, ipMask := range list {
		if ip := net.ParseIP(ipMask); ip != nil {
			ips = append(ips, &ip)
			logrus.Infoln("access ip:", ip.String())
			continue
		} else if _, ipNet, err := net.ParseCIDR(ipMask); err != nil {
			return fmt.Errorf("parsing CIDR IPs %s: %w", ipNet, err)
		} else {
			ipsNet = append(ipsNet, ipNet)
			logrus.Infoln("access CIDR:", ipNet.String())
		}
	}
	i.ips = ips
	i.ipsNet = ipsNet
	return nil
}

func (i *IPAuth) contains(addr string) bool {
	if len(addr) == 0 {
		return false
	}
	ip, err := i.parseIP(addr)
	if err != nil {
		return false
	}
	return i.containsIP(ip)
}

func (i *IPAuth) containsIP(addr net.IP) bool {
	if i.ips == nil && i.ipsNet == nil {
		return true
	}
	for _, deniedIP := range i.ips {
		if deniedIP.Equal(addr) {
			return true
		}
	}

	for _, denyNet := range i.ipsNet {
		if denyNet.Contains(addr) {
			return true
		}
	}

	return false
}

func (i *IPAuth) parseIP(addr string) (net.IP, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, fmt.Errorf("unable parse IP from address %s", addr)
	}
	return ip, nil
}
