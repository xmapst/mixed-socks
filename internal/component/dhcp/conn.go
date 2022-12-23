package dhcp

import (
	"context"
	"github.com/xmapst/mixed-socks/internal/component/dialer"
	"net"
	"runtime"
)

func ListenDHCPClient(ctx context.Context) (net.PacketConn, error) {
	listenAddr := "0.0.0.0:68"
	if runtime.GOOS == "linux" || runtime.GOOS == "android" {
		listenAddr = "255.255.255.255:68"
	}

	return dialer.ListenPacket(ctx, "udp4", listenAddr, dialer.WithAddrReuse(true))
}
