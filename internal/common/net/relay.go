package net

import (
	"io"
	"net"
	"time"
)

// Relay copies between left and right bidirectionally.
func Relay(leftConn, rightConn net.Conn) {
	ch := make(chan error)

	go func() {
		_, err := io.Copy(WriteOnlyWriter{Writer: leftConn}, ReadOnlyReader{Reader: rightConn})
		_ = leftConn.SetReadDeadline(time.Now())
		ch <- err
	}()

	_, _ = io.Copy(WriteOnlyWriter{Writer: rightConn}, ReadOnlyReader{Reader: leftConn})
	_ = rightConn.SetReadDeadline(time.Now())
	<-ch
}
