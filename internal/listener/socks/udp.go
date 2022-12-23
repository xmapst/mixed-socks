package socks

import (
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/adapter/inbound"
	"github.com/xmapst/mixed-socks/internal/common/pool"
	"github.com/xmapst/mixed-socks/internal/common/sockopt"
	"github.com/xmapst/mixed-socks/internal/constant"
	authStore "github.com/xmapst/mixed-socks/internal/listener/auth"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
)

type UDPListener struct {
	packetConn net.PacketConn
	addr       string
	closed     bool
}

// RawAddress implements constant.Listener
func (l *UDPListener) RawAddress() string {
	return l.addr
}

// Address implements constant.Listener
func (l *UDPListener) Address() string {
	return l.packetConn.LocalAddr().String()
}

// Close implements constant.Listener
func (l *UDPListener) Close() error {
	l.closed = true
	return l.packetConn.Close()
}

func NewUDP(addr string, in chan<- *inbound.PacketAdapter) (*UDPListener, error) {
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	if err := sockopt.UDPReuseaddr(l.(*net.UDPConn)); err != nil {
		logrus.Warnln("Failed to Reuse UDP Address: %s", err)
	}

	sl := &UDPListener{
		packetConn: l,
		addr:       addr,
	}
	go func() {
		for {
			buf := pool.Get(pool.UDPBufferSize)
			n, remoteAddr, err := l.ReadFrom(buf)
			if err != nil {
				_ = pool.Put(buf)
				if sl.closed {
					break
				}
				continue
			}
			if forbidden(remoteAddr) {
				continue
			}
			handleSocksUDP(l, in, buf[:n], remoteAddr)
		}
	}()

	return sl, nil
}

func forbidden(addr net.Addr) bool {
	if authStore.Whitelist() != nil {
		client := addr.String()
		if !authStore.Whitelist().Verify(client) {
			logrus.Warnf("[UDP] %s reject", client)
			return true
		}
	}
	return false
}

func handleSocksUDP(pc net.PacketConn, in chan<- *inbound.PacketAdapter, buf []byte, addr net.Addr) {
	target, payload, err := socks5.DecodeUDPPacket(buf)
	if err != nil {
		// Unresolved UDP packet, return buffer to the pool
		_ = pool.Put(buf)
		return
	}
	packet := &packet{
		pc:      pc,
		rAddr:   addr,
		payload: payload,
		bufRef:  buf,
	}
	select {
	case in <- inbound.NewPacket(target, packet, constant.SOCKS5):
	default:
	}
}
