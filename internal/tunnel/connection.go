package tunnel

import (
	"errors"
	N "github.com/xmapst/mixed-socks/internal/common/net"
	"github.com/xmapst/mixed-socks/internal/common/pool"
	"github.com/xmapst/mixed-socks/internal/constant"
	"net"
	"net/netip"
	"time"
)

func handleUDPToRemote(packet constant.UDPPacket, pc constant.PacketConn, metadata *constant.Metadata) error {
	defer packet.Drop()

	addr := metadata.UDPAddr()
	if addr == nil {
		return errors.New("udp addr invalid")
	}

	if _, err := pc.WriteTo(packet.Data(), addr); err != nil {
		return err
	}
	// reset timeout
	_ = pc.SetReadDeadline(time.Now().Add(udpTimeout))

	return nil
}

func handleUDPToLocal(packet constant.UDPPacket, pc net.PacketConn, key string, oAddr netip.Addr) {
	buf := pool.Get(pool.UDPBufferSize)
	defer func(buf []byte) {
		_ = pool.Put(buf)
	}(buf)
	defer natTable.Delete(key)
	defer func(pc net.PacketConn) {
		_ = pc.Close()
	}(pc)

	for {
		_ = pc.SetReadDeadline(time.Now().Add(udpTimeout))
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}

		fromUDPAddr := from.(*net.UDPAddr)
		_, err = packet.WriteBack(buf[:n], fromUDPAddr)
		if err != nil {
			return
		}
	}
}

func handleSocket(ctx constant.ConnContext, outbound net.Conn) {
	N.Relay(ctx.Conn(), outbound)
}
