package context

import (
	"github.com/gofrs/uuid"
	"github.com/miekg/dns"
	"net"
)

const (
	DNSTypeHost  = "host"
	DNSTypeRaw   = "raw"
	DNSTypeCache = "cache"
)

type DNSContext struct {
	id         uuid.UUID
	remoteAddr net.Addr
	localAddr  net.Addr
	msg        *dns.Msg
	tp         string
}

func NewDNSContext(localAddr, remoteAddr net.Addr, msg *dns.Msg) *DNSContext {
	id, _ := uuid.NewV4()
	return &DNSContext{
		id:         id,
		msg:        msg,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

// ID implement constant.PlainContext ID
func (c *DNSContext) ID() uuid.UUID {
	return c.id
}

// SetType set type of response
func (c *DNSContext) SetType(tp string) {
	c.tp = tp
}

// Type return type of response
func (c *DNSContext) Type() string {
	return c.tp
}

// LocalAddr returns the net.Addr of the server
func (c *DNSContext) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the net.Addr of the client that sent the current request.
func (c *DNSContext) RemoteAddr() net.Addr {
	return c.remoteAddr
}
