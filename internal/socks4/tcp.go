package socks4

import (
	"bytes"
	"encoding/binary"
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
}

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
socks4 protocol
request
byte | 0  | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | ...  |
     |0x04|cmd| port  |     ip        |  user\0  |
reply
byte | 0  |  1   | 2 | 3 | 4 | 5 | 6 | 7|
     |0x00|status|       |              |
socks4a protocol
request
byte | 0  | 1 | 2 | 3 |4 | 5 | 6 | 7 | 8 | ... |...     |
     |0x04|cmd| port  |  0.0.0.x     |  user\0 |domain\0|
reply
byte | 0  |  1  | 2 | 3 | 4 | 5 | 6| 7 |
	 |0x00|staus| port  |    ip        |
*/

func Handle(src net.Conn, buf []byte, n int, auth auth.Service) net.Conn {
	d := net.Dialer{Timeout: conf.App.ParseTimeout()}
	guid := xid.New()
	p := &proxy{
		src:  src,
		auth: auth,
		log: logrus.WithFields(logrus.Fields{
			"uuid": guid.String(),
		}),
		dial: d.Dial,
	}
	p.log = p.log.WithField("src", p.srcAddr())
	p.log.Info("request service using socks4 protocol")
	target, err := p.handshake(buf, n)
	if err != nil {
		return p.dest
	}
	p.processRequest(target)
	return p.dest
}

func (p *proxy) handshake(buf []byte, n int) (target string, err error) {
	if n < 8 {
		n1, err := io.ReadAtLeast(p.src, buf[n:], 8-n)
		if err != nil {
			p.log.Errorln(ErrRequestRejected, err)
			return "", ErrRequestRejected
		}
		n += n1
	}
	buf = buf[1:n]
	command := buf[0]
	p.log = p.log.WithField("command", cmdMap[command])
	// command only support connect
	if command != CmdConnect {
		logrus.Errorln(ErrRequestUnknownCode)
		return "", ErrRequestUnknownCode
	}
	user := p.readUntilNull(buf[7:])
	if p.auth.Enable() {
		ret := p.auth.Verify(
			user, "",
		)
		if !ret {
			_, _ = p.src.Write([]byte{0x01, 0x00})
			p.log.Errorln(ErrRequestIdentdMismatched)
			return "", ErrRequestIdentdMismatched
		}
	}
	// get port
	port := binary.BigEndian.Uint16(buf[1:3])

	// get ip
	ip := net.IP(buf[3:7])

	// NULL-terminated user string
	// jump to NULL character
	var j int
	for j = 7; j < n-1; j++ {
		if buf[j] == 0x00 {
			break
		}
	}

	host := ip.String()

	// socks4a
	// 0.0.0.x
	if ip[0] == 0x00 && ip[1] == 0x00 && ip[2] == 0x00 && ip[3] != 0x00 {
		j++
		var i = j

		// jump to the end of hostname
		for j = i; j < n-1; j++ {
			if buf[j] == 0x00 {
				break
			}
		}
		host = string(buf[i:j])
	}

	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func (p *proxy) processRequest(target string) {
	var err error
	p.log = p.log.WithField("dest", target)
	p.log.Info("establish connection")
	// connect to the target
	p.dest, err = p.dial("tcp", target)
	if err != nil {
		// connection failed
		_, _ = p.src.Write([]byte{0x00, 0x5b, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00})
		p.log.Errorln(ErrRequestIdentdFailed, err)
		return
	}
	p.log = p.log.WithField("proxy", p.proxyAddr())
	p.log = p.log.WithField("dest", p.destAddr())
	_, err = p.src.Write([]byte{0x00, 0x5A, 0x00, 0x00, 0, 0, 0, 0})
	if err != nil {
		p.log.Errorln("write  response error", err)
		return
	}

	p.log.Infoln("connection established")
	common.Forward(p.src, p.dest)
}

func (p *proxy) readUntilNull(src []byte) string {
	buf := &bytes.Buffer{}
	for _, v := range src {
		if v == 0 {
			break
		}
		buf.WriteByte(v)
	}
	return buf.String()
}
