package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"github/xmapst/mixed-socks/internal/conf"
	"io"
	"net"
)

type proxy struct {
	log  *logrus.Entry
	src  net.Conn
	dest net.Conn
	auth auth.Service
	dial common.DialFunc
	udp  string
	port int
}

type DialFunc func(network, addr string) (net.Conn, error)

func (p *proxy) srcAddr() string {
	if p.src != nil {
		return p.src.RemoteAddr().String()
	}
	return ""
}

func (p *proxy) proxyAddr() string {
	if p.dest != nil {
		return p.dest.LocalAddr().String()
	}
	return ""
}

func (p *proxy) destAddr() string {
	if p.dest != nil {
		return p.dest.RemoteAddr().String()
	}
	return ""
}

/*
socks5 protocol
initial
byte | 0  |   1    | 2 | ...... | n |
     |0x05|num auth|  auth methods  |
reply
byte | 0  |  1  |
     |0x05| auth|
username/password auth request
byte | 0  |  1         |          |     1 byte   |          |
     |0x01|username_len| username | password_len | password |
username/password auth reponse
byte | 0  | 1    |
     |0x01|status|
request
byte | 0  | 1 | 2  |   3    | 4 | .. | n-2 | n-1| n |
     |0x05|cmd|0x00|addrtype|      addr    |  port  |
response
byte |0   |  1   | 2  |   3    | 4 | .. | n-2 | n-1 | n |
     |0x05|status|0x00|addrtype|     addr     |  port   |
*/

func Handle(src net.Conn, buf []byte, n int, auth auth.Service, udpAddr string, port int) net.Conn {
	d := net.Dialer{Timeout: conf.App.ParseTimeout()}
	guid := xid.New()
	p := &proxy{
		src:  src,
		auth: auth,
		log: logrus.WithFields(logrus.Fields{
			"uud": guid.String(),
		}),
		udp:  udpAddr,
		port: port,
		dial: d.Dial,
	}
	p.log = p.log.WithField("src", p.srcAddr())
	if err := p.handshake(buf, n); err != nil {
		return p.dest
	}
	p.processRequest()
	return p.dest
}

func (p *proxy) handshake(buf []byte, n int) error {
	// read auth methods
	if n < 2 {
		n1, err := io.ReadAtLeast(p.src, buf[1:], 1)
		if err != nil {
			p.log.Errorln(err)
			return err
		}
		n += n1
	}

	l := int(buf[1])
	if n != (l + 2) {
		// read remains data
		n1, err := io.ReadFull(p.src, buf[n:l+2+1])
		if err != nil {
			p.log.Errorln(err)
			return err
		}
		n += n1
	}

	if !p.auth.Enable() {
		// no auth required
		_, _ = p.src.Write([]byte{0x05, 0x00})
		return nil
	}

	hasPassAuth := false
	var passAuth byte = 0x02

	// check auth method
	// only password(0x02) supported
	for i := 2; i < n; i++ {
		if buf[i] == passAuth {
			hasPassAuth = true
			break
		}
	}

	if !hasPassAuth {
		_, _ = p.src.Write([]byte{0x05, 0xff})
		p.log.Errorln("no supported auth method")
		return errors.New("no supported auth method")
	}

	return p.passwordAuth()
}

func (p *proxy) passwordAuth() error {
	buf := make([]byte, 32)

	// username/password required
	_, _ = p.src.Write([]byte{0x05, 0x02})
	n, err := io.ReadAtLeast(p.src, buf, 2)
	if err != nil {
		p.log.Errorln(err)
		return err
	}
	// check auth version
	if buf[0] != 0x01 {
		p.log.Errorln("unsupported auth version")
		return errors.New("unsupported auth version")
	}

	usernameLen := int(buf[1])
	p0 := 2
	p1 := p0 + usernameLen
	for n < p1 {
		var n1 int
		n1, err = p.src.Read(buf[n:])
		if err != nil {
			p.log.Errorln(err)
			return err
		}
		n += n1
	}
	user := string(buf[p0:p1])
	p.log = p.log.WithField("user", user)
	passwordLen := int(buf[p1])

	p3 := p1 + 1
	p4 := p3 + passwordLen

	for n < p4 {
		var n1 int
		n1, err = p.src.Read(buf[n:])
		if err != nil {
			p.log.Errorln(err)
			return err
		}
		n += n1
	}

	password := buf[p3:p4]
	if p.auth.Enable() {
		ret := p.auth.Verify(
			user, string(password),
		)
		if ret {
			_, _ = p.src.Write([]byte{0x01, 0x00})
			return nil
		}
		_, _ = p.src.Write([]byte{0x01, 0x01})
		p.log.Errorln("access denied")
		return errors.New("access denied")
	}
	p.log.Errorln("authentication failed")
	return errors.New("authentication failed")
}

func (p *proxy) processRequest() {
	buf := make([]byte, 258)

	// read header
	n, err := io.ReadAtLeast(p.src, buf, 10)
	if err != nil {
		p.log.Errorln(err)
		return
	}

	if buf[0] != Version {
		p.log.Errorln("error version", buf[0])
		return
	}

	hlen := 0   // target address length
	host := ""  // target address
	msglen := 0 // header length

	switch buf[3] {
	case common.AtypIPv4:
		hlen = 4
	case common.AtypDomainName:
		hlen = int(buf[4]) + 1
	case common.AtypIPv6:
		hlen = 16
	}

	msglen = 6 + hlen

	if n < msglen {
		// read remains header
		_, err = io.ReadFull(p.src, buf[n:msglen])
		if err != nil {
			p.log.Errorln(err)
			return
		}
	}

	// get target address
	addr := buf[4 : 4+hlen]
	if buf[3] == common.AtypDomainName {
		host = string(addr[1:])
	} else {
		host = net.IP(addr).String()
	}

	// get target port
	port := binary.BigEndian.Uint16(buf[msglen-2 : msglen])

	// target address
	target := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	p.log = p.log.WithField("command", cmdMap[buf[1]])
	// command support connect
	switch buf[1] {
	case CmdUdp:
		p.handleUdpCmd()
	case CmdConnect, CmdBind:
		p.handleConnectCmd(target)
	default:
		return
	}
}

func (p *proxy) handleConnectCmd(target string) {
	var err error
	p.log = p.log.WithField("dest", target)
	p.log.Info("establish connection")
	// connect to the target
	p.dest, err = p.dial("tcp", target)
	p.log.Info("request service using socks5 protocol")
	if err != nil {
		// connection failed
		_, _ = p.src.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})
		p.log.Errorln(err)
		return
	}

	p.log = p.log.WithField("proxy", p.proxyAddr())
	p.log = p.log.WithField("dest", p.destAddr())
	// connection success
	_, _ = p.src.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})
	p.log.Infoln("connection established")
	// enter data exchange
	common.Forward(p.src, p.dest)
	return
}

func (p *proxy) handleUdpCmd() {
	udpAddr, _ := net.ResolveIPAddr("ip", p.udp)
	hostByte := udpAddr.IP.To4()
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(p.port))
	buf := append([]byte{Version, 0x00, 0x00, 0x01}, hostByte...)
	buf = append(buf, portByte...)
	_, err := p.src.Write(buf)
	if err != nil {
		p.log.Errorln("write response error", err)
		return
	}

	forward := func(src net.Conn) {
		defer func(src net.Conn) {
			_ = src.Close()
		}(src)
		for {
			_, err = io.ReadFull(src, make([]byte, 100))
			if err != nil {
				p.log.Errorln(err)
				break
			}
		}
	}

	go forward(p.src)
	return
}
