package common

import (
	"github/xmapst/mixed-socks/internal/statistic"
	"io"
	"net"
)

type DialFunc func(network, addr string) (net.Conn, error)

func Forward(id string, src, dest net.Conn, metadata *statistic.Metadata) {
	src = statistic.NewTCPTracker(id, src, metadata)
	defer func(src, dest net.Conn) {
		_ = dest.Close()
		_ = src.Close()
	}(src, dest)
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(src, dest)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(dest, src)
		done <- struct{}{}
	}()
	<-done
}
