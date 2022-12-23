package resolver

import (
	"net"
)

var DefaultHostMapper Enhancer

type Enhancer interface {
	MappingEnabled() bool
	FindHostByIP(net.IP) (string, bool)
}

func MappingEnabled() bool {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.MappingEnabled()
	}

	return false
}

func FindHostByIP(ip net.IP) (string, bool) {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.FindHostByIP(ip)
	}

	return "", false
}
