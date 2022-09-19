package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"io"
	"net"
	"strconv"
)

type Proxy struct {
	Log  *logrus.Entry
	Src  net.Conn
	Dest net.Conn
	Auth auth.Authenticator
	Dial common.DialFunc
	Udp  string
}

type DialFunc func(network, addr string) (net.Conn, error)

func (p *Proxy) srcAddr() string {
	if p.Src != nil {
		return p.Src.RemoteAddr().String()
	}
	return ""
}

func (p *Proxy) proxyAddr() string {
	if p.Dest != nil {
		return p.Dest.LocalAddr().String()
	}
	return ""
}

func (p *Proxy) destAddr() string {
	if p.Dest != nil {
		return p.Dest.RemoteAddr().String()
	}
	return ""
}

func (p *Proxy) SrcConn() net.Conn {
	return p.Src
}
func (p *Proxy) DestConn() net.Conn {
	return p.Dest
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

func (p *Proxy) Handle(buf []byte, n int) {
	p.Log = p.Log.WithField("src", p.srcAddr())
	if err := p.handshake(buf, n); err != nil {
		return
	}
	p.processRequest()
	return
}

func (p *Proxy) handshake(buf []byte, n int) error {
	// read auth methods
	if n < 2 {
		n1, err := io.ReadAtLeast(p.Src, buf[1:], 1)
		if err != nil {
			p.Log.Errorln(err)
			return err
		}
		n += n1
	}

	l := int(buf[1])
	if n != (l + 2) {
		// read remains data
		n1, err := io.ReadFull(p.Src, buf[n:l+2+1])
		if err != nil {
			p.Log.Errorln(err)
			return err
		}
		n += n1
	}

	if !p.Auth.Enable() {
		// no auth required
		_, _ = p.Src.Write([]byte{0x05, 0x00})
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
		_, _ = p.Src.Write([]byte{0x05, 0xff})
		p.Log.Errorln("no supported auth method")
		return errors.New("no supported auth method")
	}

	return p.passwordAuth()
}

func (p *Proxy) passwordAuth() error {
	buf := make([]byte, 32)

	// username/password required
	_, _ = p.Src.Write([]byte{0x05, 0x02})
	n, err := io.ReadAtLeast(p.Src, buf, 2)
	if err != nil {
		p.Log.Errorln(err)
		return err
	}
	// check auth version
	if buf[0] != 0x01 {
		p.Log.Errorln("unsupported auth version")
		return errors.New("unsupported auth version")
	}

	usernameLen := int(buf[1])
	p0 := 2
	p1 := p0 + usernameLen
	for n < p1 {
		var n1 int
		n1, err = p.Src.Read(buf[n:])
		if err != nil {
			p.Log.Errorln(err)
			return err
		}
		n += n1
	}
	user := string(buf[p0:p1])
	p.Log = p.Log.WithField("user", user)
	passwordLen := int(buf[p1])

	p3 := p1 + 1
	p4 := p3 + passwordLen

	for n < p4 {
		var n1 int
		n1, err = p.Src.Read(buf[n:])
		if err != nil {
			p.Log.Errorln(err)
			return err
		}
		n += n1
	}

	password := buf[p3:p4]
	if p.Auth.Verify(user, string(password), p.srcAddr()) {
		_, _ = p.Src.Write([]byte{0x01, 0x00})
		return nil
	}
	_, _ = p.Src.Write([]byte{0x01, 0x01})
	p.Log.Errorln("access denied")
	return errors.New("access denied")
}

func (p *Proxy) processRequest() {
	buf := make([]byte, 258)

	// read header
	n, err := io.ReadAtLeast(p.Src, buf, 10)
	if err != nil {
		p.Log.Errorln(err)
		return
	}

	if buf[0] != Version {
		p.Log.Errorln("error version", buf[0])
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
		_, err = io.ReadFull(p.Src, buf[n:msglen])
		if err != nil {
			p.Log.Errorln(err)
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
	p.Log = p.Log.WithField("command", cmdMap[buf[1]])
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

func (p *Proxy) handleConnectCmd(target string) {
	var err error
	p.Log = p.Log.WithField("dest", target)
	p.Log.Info("establish connection")
	// connect to the target
	p.Dest, err = p.Dial("tcp", target)
	if err != nil {
		// connection failed
		_, _ = p.Src.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})
		p.Log.Errorln(err)
		return
	}

	p.Log = p.Log.WithField("proxy", p.proxyAddr())
	p.Log = p.Log.WithField("dest", p.destAddr())
	// connection success
	_, _ = p.Src.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})
	p.Log.Infoln("connection established")
	// enter data exchange
	common.Forward(p.Src, p.Dest)
	return
}

func (p *Proxy) handleUdpCmd() {
	_, port, err := net.SplitHostPort(p.Udp)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	_port, err := strconv.Atoi(port)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	udpAddr, _ := net.ResolveIPAddr("ip", p.Udp)
	hostByte := udpAddr.IP.To4()
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(_port))
	buf := append([]byte{Version, 0x00, 0x00, 0x01}, hostByte...)
	buf = append(buf, portByte...)
	_, err = p.Src.Write(buf)
	if err != nil {
		p.Log.Errorln("write response error", err)
		return
	}

	forward := func(src net.Conn) {
		defer func(src net.Conn) {
			_ = src.Close()
		}(src)
		for {
			_, err = io.ReadFull(src, make([]byte, 100))
			if err != nil {
				p.Log.Errorln(err)
				break
			}
		}
	}

	go forward(p.Src)
	return
}
