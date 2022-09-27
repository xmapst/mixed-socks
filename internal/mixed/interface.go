package mixed

import (
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"github/xmapst/mixed-socks/internal/http"
	"github/xmapst/mixed-socks/internal/socks4"
	"github/xmapst/mixed-socks/internal/socks5"
	"net"
)

type Proxy interface {
	Handle(uuid string, conn net.Conn, authenticator auth.Authenticator, dial common.DialFunc, log *logrus.Entry)
}

func newSocks4() Proxy {
	return &socks4.Proxy{}
}

func newSocks5(udpAddr string) Proxy {
	return &socks5.Proxy{Udp: udpAddr}
}

func newHttp() Proxy { return &http.Proxy{} }
