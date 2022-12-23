package dialer

import (
	"golang.org/x/sys/windows"
	"net"
	"syscall"
)

func addrReuseToListenConfig(lc *net.ListenConfig) {
	chain := lc.Control

	lc.Control = func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		return c.Control(func(fd uintptr) {
			_ = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
		})
	}
}
