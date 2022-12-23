package socks

import (
	"github.com/xmapst/mixed-socks/internal/adapter/inbound"
	"github.com/xmapst/mixed-socks/internal/constant"
	authStore "github.com/xmapst/mixed-socks/internal/listener/auth"
	"github.com/xmapst/mixed-socks/internal/transport/socks4"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"io"
	"net"
)

type Listener struct {
	listener net.Listener
	addr     string
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

func HandleSocks4(conn net.Conn, in chan<- constant.ConnContext) {
	addr, _, err := socks4.ServerHandshake(conn, authStore.Authenticator())
	if err != nil {
		_ = conn.Close()
		return
	}
	in <- inbound.NewSocket(socks5.ParseAddr(addr), conn, constant.SOCKS4)
}

func HandleSocks5(conn net.Conn, in chan<- constant.ConnContext) {
	target, command, err := socks5.ServerHandshake(conn, authStore.Authenticator())
	if err != nil {
		_ = conn.Close()
		return
	}
	if command == socks5.CmdUDPAssociate {
		defer func(conn net.Conn) {
			_ = conn.Close()
		}(conn)
		_, _ = io.Copy(io.Discard, conn)
		return
	}
	in <- inbound.NewSocket(target, conn, constant.SOCKS5)
}
