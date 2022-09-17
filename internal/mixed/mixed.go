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
	"log"
	"net"
	"sync"
	"time"
)

type Listener struct {
	tcp    net.Listener
	udp    *net.UDPConn
	addr   string
	host   string
	port   int
	auth   auth.Service
	closed bool
	wg     *sync.WaitGroup
}

func (l *Listener) RawAddress() string {
	return l.addr
}

func (l *Listener) Address() string {
	return l.tcp.Addr().String()
}

func (l *Listener) close() {
	l.closed = l.tcp.Close() == nil && l.udp.Close() == nil
	return
}

func (l *Listener) Running() bool {
	return !l.closed
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return nil
	}
}

func New(ctx context.Context, host string, port int, ip auth.Service) (ml *Listener, err error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	ml = &Listener{
		host: host,
		port: port,
		addr: addr,
		auth: ip,
		wg:   &sync.WaitGroup{},
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	lc := net.ListenConfig{}
	ml.tcp, err = lc.Listen(ctx, "tcp", tcpAddr.String())
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	ml.udp, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	return ml, err
}

func (l *Listener) ListenAndServe(auth auth.Service) {
	// udp
	go func() {
		logrus.Infoln("listen udp", l.udp.LocalAddr().String())
		log.Println("Listen udp", l.udp.LocalAddr().String())
		udp.Listen(l.udp)
	}()
	// tcp
	listenAddr := []string{
		fmt.Sprintf("http://%s", l.Address()),
		fmt.Sprintf("socks4://%s", l.Address()),
		fmt.Sprintf("socks5://%s", l.Address()),
	}
	logrus.Infoln("listen tcp", listenAddr)
	log.Println("Listen tcp", listenAddr)
	ln := &proxyproto.Listener{Listener: l.tcp}
	for {
		if l.closed {
			break
		}
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		clientIP := conn.RemoteAddr().String()
		if !l.auth.Verify(clientIP) {
			logrus.Warningln(clientIP, "access denied, not in allowed address group")
			_ = conn.Close()
		} else {
			l.wg.Add(1)
			go l.handle(conn, auth)
		}
	}
}

func (l *Listener) handle(src net.Conn, auth auth.Service) {
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
		dest = socks5.Handle(src, buf, n, auth, l.udp.LocalAddr().String(), l.port)
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
