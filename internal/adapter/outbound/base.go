package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xmapst/mixed-socks/internal/component/dialer"
	"github.com/xmapst/mixed-socks/internal/constant"
	"net"
)

type Base struct {
	name  string
	addr  string
	iface string
	tp    constant.AdapterType
	udp   bool
}

// Name implements constant.ProxyAdapter
func (b *Base) Name() string {
	return b.name
}

// Type implements constant.ProxyAdapter
func (b *Base) Type() constant.AdapterType {
	return b.tp
}

// StreamConn implements constant.ProxyAdapter
func (b *Base) StreamConn(c net.Conn, _ *constant.Metadata) (net.Conn, error) {
	return c, errors.New("no support")
}

// ListenPacketContext implements constant.ProxyAdapter
func (b *Base) ListenPacketContext(_ context.Context, _ *constant.Metadata, _ ...dialer.Option) (constant.PacketConn, error) {
	return nil, errors.New("no support")
}

// SupportUDP implements constant.ProxyAdapter
func (b *Base) SupportUDP() bool {
	return b.udp
}

// MarshalJSON implements constant.ProxyAdapter
func (b *Base) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type": b.Type().String(),
	})
}

// Addr implements constant.ProxyAdapter
func (b *Base) Addr() string {
	return b.addr
}

// Unwrap implements constant.ProxyAdapter
func (b *Base) Unwrap(_ *constant.Metadata) constant.Proxy {
	return nil
}

// DialOptions return []dialer.Option from struct
func (b *Base) DialOptions(opts ...dialer.Option) []dialer.Option {
	if b.iface != "" {
		opts = append(opts, dialer.WithInterface(b.iface))
	}

	return opts
}

type BasicOption struct {
	Interface string `proxy:"interface-name,omitempty" group:"interface-name,omitempty"`
}

type BaseOption struct {
	Name      string
	Addr      string
	Type      constant.AdapterType
	UDP       bool
	Interface string
}

type conn struct {
	net.Conn
	chain constant.Chain
}

// Chains implements constant.Connection
func (c *conn) Chains() constant.Chain {
	return c.chain
}

// AppendToChains implements constant.Connection
func (c *conn) AppendToChains(a constant.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func NewConn(c net.Conn, a constant.ProxyAdapter) constant.Conn {
	return &conn{c, []string{a.Name()}}
}

type packetConn struct {
	net.PacketConn
	chain constant.Chain
}

// Chains implements constant.Connection
func (c *packetConn) Chains() constant.Chain {
	return c.chain
}

// AppendToChains implements constant.Connection
func (c *packetConn) AppendToChains(a constant.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func newPacketConn(pc net.PacketConn, a constant.ProxyAdapter) constant.PacketConn {
	return &packetConn{pc, []string{a.Name()}}
}
