package inbound

import (
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/context"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
)

// NewSocket receive TCP inbound and return ConnContext
func NewSocket(target socks5.Addr, conn net.Conn, source constant.Type) *context.ConnContext {
	metadata := parseSocksAddr(target)
	metadata.NetWork = constant.TCP
	metadata.Type = source
	if ip, port, err := parseAddr(conn.RemoteAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}

	return context.NewConnContext(conn, metadata)
}
