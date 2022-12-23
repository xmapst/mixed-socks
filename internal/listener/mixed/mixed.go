package mixed

import (
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/common/cache"
	N "github.com/xmapst/mixed-socks/internal/common/net"
	"github.com/xmapst/mixed-socks/internal/constant"
	authStore "github.com/xmapst/mixed-socks/internal/listener/auth"
	"github.com/xmapst/mixed-socks/internal/listener/http"
	"github.com/xmapst/mixed-socks/internal/listener/socks"
	"github.com/xmapst/mixed-socks/internal/transport/socks4"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
)

type Listener struct {
	listener net.Listener
	addr     string
	cache    *cache.LruCache
	closed   bool
}

// RawAddress implements constant.Listener
func (l *Listener) RawAddress() string {
	return l.addr
}

// Address implements constant.Listener
func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

// Close implements constant.Listener
func (l *Listener) Close() error {
	l.closed = true
	return l.listener.Close()
}

func New(addr string, in chan<- constant.ConnContext) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ml := &Listener{
		listener: l,
		addr:     addr,
		cache:    cache.New(cache.WithAge(30)),
	}
	go func() {
		for {
			c, err := ml.listener.Accept()
			if err != nil {
				if ml.closed {
					break
				}
				continue
			}
			if forbidden(c) {
				continue
			}
			go handleConn(c, in, ml.cache)
		}
	}()

	return ml, nil
}

func forbidden(conn net.Conn) bool {
	if authStore.Whitelist() != nil {
		client := conn.RemoteAddr().String()
		if !authStore.Whitelist().Verify(client) {
			logrus.Warnf("[TCP] %s reject", client)
			_ = conn.Close()
			return true
		}
	}
	return false
}

func handleConn(conn net.Conn, in chan<- constant.ConnContext, cache *cache.LruCache) {
	_ = conn.(*net.TCPConn).SetKeepAlive(true)

	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		return
	}

	switch head[0] {
	case socks4.Version:
		socks.HandleSocks4(bufConn, in)
	case socks5.Version:
		socks.HandleSocks5(bufConn, in)
	default:
		http.HandleConn(bufConn, in, cache)
	}
}
