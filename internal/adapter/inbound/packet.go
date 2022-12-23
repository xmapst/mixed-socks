package inbound

import (
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
)

// PacketAdapter is a UDP Packet adapter for socks/redir/tun
type PacketAdapter struct {
	constant.UDPPacket
	metadata *constant.Metadata
}

// Metadata returns destination metadata
func (s *PacketAdapter) Metadata() *constant.Metadata {
	return s.metadata
}

// NewPacket is PacketAdapter generator
func NewPacket(target socks5.Addr, packet constant.UDPPacket, source constant.Type) *PacketAdapter {
	metadata := parseSocksAddr(target)
	metadata.NetWork = constant.UDP
	metadata.Type = source
	if ip, port, err := parseAddr(packet.LocalAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}

	return &PacketAdapter{
		UDPPacket: packet,
		metadata:  metadata,
	}
}
