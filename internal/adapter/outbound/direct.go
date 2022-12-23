package outbound

import (
	"context"
	"github.com/xmapst/mixed-socks/internal/component/dialer"
	"github.com/xmapst/mixed-socks/internal/constant"
	"net"
)

type Direct struct {
	*Base
}

// DialContext implements constant.ProxyAdapter
func (d *Direct) DialContext(ctx context.Context, metadata *constant.Metadata, opts ...dialer.Option) (constant.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", metadata.RemoteAddress(), d.Base.DialOptions(opts...)...)
	if err != nil {
		return nil, err
	}
	tcpKeepAlive(c)
	return NewConn(c, d), nil
}

// ListenPacketContext implements constant.ProxyAdapter
func (d *Direct) ListenPacketContext(ctx context.Context, _ *constant.Metadata, opts ...dialer.Option) (constant.PacketConn, error) {
	pc, err := dialer.ListenPacket(ctx, "udp", "", d.Base.DialOptions(opts...)...)
	if err != nil {
		return nil, err
	}
	return newPacketConn(&directPacketConn{pc}, d), nil
}

type directPacketConn struct {
	net.PacketConn
}

func NewDirect() *Direct {
	return &Direct{
		Base: &Base{
			name: "DIRECT",
			tp:   constant.Direct,
			udp:  true,
		},
	}
}
