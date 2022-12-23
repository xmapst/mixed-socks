package inbound

import (
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/context"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
)

// NewHTTP receive normal http request and return HTTPContext
func NewHTTP(target socks5.Addr, source net.Addr, conn net.Conn) *context.ConnContext {
	metadata := parseSocksAddr(target)
	metadata.NetWork = constant.TCP
	metadata.Type = constant.HTTP
	if ip, port, err := parseAddr(source.String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}
	return context.NewConnContext(conn, metadata)
}
