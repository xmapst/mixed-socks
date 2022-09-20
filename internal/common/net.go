package common

import (
	"io"
	"net"
)

type DialFunc func(network, addr string) (net.Conn, error)

func Forward(src, dest net.Conn) {
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
