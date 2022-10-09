package mixed

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"github/xmapst/mixed-socks/internal/service"
	"github/xmapst/mixed-socks/internal/socks4"
	"github/xmapst/mixed-socks/internal/socks5"
	"github/xmapst/mixed-socks/internal/udp"
	"net"
	"sync"
	"time"
)

type Listener struct {
	tcp    net.Listener
	udp    *net.UDPConn
	host   string
	port   int64
	closed bool
	wg     *sync.WaitGroup
	conf   *service.Conf
	auth   auth.Authenticator
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

func (l *Listener) State() bool {
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
		time.Sleep(100 * time.Millisecond)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return nil
	}
}

func New() *Listener {
	_conf := &service.Conf{
		Host:    common.DefaultHost,
		Port:    common.DefaultPort,
		Timeout: common.DefaultTimeout,
	}
	res := _conf.Get()
	return &Listener{
		host:   res.Host,
		port:   res.Port,
		closed: true,
		conf:   _conf,
		auth:   new(auth.Auth),
	}
}

func (l *Listener) ListenAndServe() (err error) {
	time.Sleep(100 * time.Millisecond)
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
	l.closed = false
	// udp
	go func() {
		logrus.Infoln("UDP Server Listening At:", l.udp.LocalAddr().String())
		udp.Listen(l.udp)
	}()
	// tcp
	listenAddr := []string{
		fmt.Sprintf("http://%s", l.Address()),
		fmt.Sprintf("socks4://%s", l.Address()),
		fmt.Sprintf("socks5://%s", l.Address()),
	}
	logrus.Infoln("TCP Server Listening At:", listenAddr)
	go func() {
		ln := &proxyproto.Listener{Listener: l.tcp}
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
				go l.handle(conn)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (l *Listener) handle(conn net.Conn) {
	defer l.wg.Done()
	bufConn := common.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	d := net.Dialer{Timeout: l.conf.Get().ParseTimeout()}
	var proxy Proxy
	switch head[0] {
	case socks4.Version:
		proxy = newSocks4()
	case socks5.Version:
		proxy = newSocks5(l.udp.LocalAddr().String())
	default:
		proxy = newHttp()
	}
	id, _ := uuid.NewV4()
	log := logrus.WithFields(logrus.Fields{
		"uuid": id,
	})
	proxy.Handle(id, bufConn, l.auth, d.Dial, log)
}
