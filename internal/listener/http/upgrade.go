package http

import (
	"github.com/xmapst/mixed-socks/internal/adapter/inbound"
	N "github.com/xmapst/mixed-socks/internal/common/net"
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
	"net/http"
	"strings"
)

func isUpgradeRequest(req *http.Request) bool {
	for _, header := range req.Header["Connection"] {
		for _, elm := range strings.Split(header, ",") {
			if strings.EqualFold(strings.TrimSpace(elm), "Upgrade") {
				return true
			}
		}
	}

	return false
}

func handleUpgrade(conn net.Conn, request *http.Request, in chan<- constant.ConnContext) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	removeProxyHeaders(request.Header)
	removeExtraHTTPHostPort(request)

	address := request.Host
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "80")
	}

	dstAddr := socks5.ParseAddr(address)
	if dstAddr == nil {
		return
	}

	left, right := net.Pipe()

	in <- inbound.NewHTTP(dstAddr, conn.RemoteAddr(), right)

	bufferedLeft := N.NewBufferedConn(left)
	defer func(bufferedLeft *N.BufferedConn) {
		_ = bufferedLeft.Close()

	}(bufferedLeft)

	err := request.Write(bufferedLeft)
	if err != nil {
		return
	}

	resp, err := http.ReadResponse(bufferedLeft.Reader(), request)
	if err != nil {
		return
	}

	removeProxyHeaders(resp.Header)

	err = resp.Write(conn)
	if err != nil {
		return
	}

	if resp.StatusCode == http.StatusSwitchingProtocols {
		N.Relay(bufferedLeft, conn)
	}
}
