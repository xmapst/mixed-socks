package mixed

import (
	"context"
	"errors"
	"fmt"
	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/http"
	"github/xmapst/mixed-socks/internal/socks4"
	"github/xmapst/mixed-socks/internal/socks5"
	"github/xmapst/mixed-socks/internal/udp"
	"io"
	"net"
	"sync"
	"time"
)

type Listener struct {
	tcp    net.Listener
	udp    *net.UDPConn
	host   string
	port   int
	closed bool
	wg     *sync.WaitGroup
}

func (l *Listener) RawAddress() string {
	return fmt.Sprintf("%s:%d", l.host, l.port)
}

func (l *Listener) Address() string {
	return l.tcp.Addr().String()
}

func (l *Listener) close() {
	if l.tcp != nil && l.udp != nil {
		l.closed = l.tcp.Close() == nil && l.udp.Close() == nil
		return
	}
	if l.tcp != nil {
		l.closed = l.tcp.Close() == nil
		return
	}
	if l.udp != nil {
		l.closed = l.udp.Close() == nil
		return
	}
	return
}

func (l *Listener) Running() bool {
	return !l.closed && l.tcp != nil && l.udp != nil
}

func (l *Listener) Shutdown() error {
	return l.ShutdownWithTimeout(15 * time.Second)
}

func (l *Listener) ShutdownWithTimeout(timeout time.Duration) error {
	l.close()
	if !l.closed {
		return errors.New("shutdown error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := make(chan struct{})
	go func() {
		defer close(c)
		l.wg.Wait()
	}()
	defer func() {
		logrus.Infoln("server closed")
		l.tcp = nil
		l.udp = nil
		time.Sleep(1 * time.Second)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return nil
	}
}

func New(host string, port int) *Listener {
	return &Listener{
		host:   host,
		port:   port,
		closed: true,
	}
}

func (l *Listener) ListenAndServe() (err error) {
	l.wg = &sync.WaitGroup{}
	tcpAddr, err := net.ResolveTCPAddr("tcp", l.RawAddress())
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	l.tcp, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", l.RawAddress())
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	l.udp, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	// udp
	go func() {
		logrus.Infoln("listen udp", l.udp.LocalAddr().String())
		udp.Listen(l.udp)
	}()
	// tcp
	listenAddr := []string{
		fmt.Sprintf("http://%s", l.Address()),
		fmt.Sprintf("socks4://%s", l.Address()),
		fmt.Sprintf("socks5://%s", l.Address()),
	}
	logrus.Infoln("listen tcp", listenAddr)
	ln := &proxyproto.Listener{Listener: l.tcp}
	l.closed = false
	for {
		if l.closed {
			break
		}
		var conn net.Conn
		conn, err = ln.Accept()
		if err != nil {
			continue
		}
		clientIP := conn.RemoteAddr().String()
		if !auth.VerifyIP(clientIP) {
			logrus.Warningln(clientIP, "access denied, not in allowed address group")
			_ = conn.Close()
		} else {
			l.wg.Add(1)
			go l.handle(conn, &auth.User{})
		}
	}
	return nil
}

func (l *Listener) handle(src net.Conn, auth auth.Authenticator) {
	defer l.wg.Done()
	buf := make([]byte, 512)
	// read version
	n, err := io.ReadAtLeast(src, buf, 1)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	var dest net.Conn
	ver := buf[0]
	switch ver {
	case socks4.Version:
		dest = socks4.Handle(src, buf, n, auth)
	case socks5.Version:
		dest = socks5.Handle(src, buf, n, auth, l.udp.LocalAddr().String())
	default:
		dest = http.Handle(src, buf, auth)
	}
	if src != nil {
		_ = src.Close()
	}
	if dest != nil {
		_ = dest.Close()
	}
}
