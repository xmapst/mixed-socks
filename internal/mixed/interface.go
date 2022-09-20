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
	Handle([]byte, int)
	SrcConn() net.Conn
	DestConn() net.Conn
}

func newSocks4(src net.Conn, auth auth.Authenticator, log *logrus.Entry, dialFunc common.DialFunc) Proxy {
	return &socks4.Proxy{
		Src:  src,
		Auth: auth,
		Log:  log,
		Dial: dialFunc,
	}
}

func newSocks5(src net.Conn, auth auth.Authenticator, log *logrus.Entry, dialFunc common.DialFunc, udpAddr string) Proxy {
	return &socks5.Proxy{
		Src:  src,
		Auth: auth,
		Log:  log,
		Dial: dialFunc,
		Udp:  udpAddr,
	}
}

func newHttp(src net.Conn, auth auth.Authenticator, log *logrus.Entry, dialFunc common.DialFunc) Proxy {
	return &http.Proxy{
		Src:  src,
		Auth: auth,
		Log:  log,
		Dial: dialFunc,
	}
}
