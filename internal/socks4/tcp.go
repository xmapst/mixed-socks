package socks4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"io"
	"net"
)

type Proxy struct {
	Log  *logrus.Entry
	Src  net.Conn
	Dest net.Conn
	Auth auth.Authenticator
	Dial common.DialFunc
}

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

func (p *Proxy) Handle(buf []byte, n int) {
	p.Log = p.Log.WithField("src", p.srcAddr())
	target, err := p.handshake(buf, n)
	if err != nil {
		return
	}
	p.processRequest(target)
	return
}

func (p *Proxy) handshake(buf []byte, n int) (target string, err error) {
	if n < 8 {
		n1, err := io.ReadAtLeast(p.Src, buf[n:], 8-n)
		if err != nil {
			p.Log.Errorln(ErrRequestRejected, err)
			return "", ErrRequestRejected
		}
		n += n1
	}
	buf = buf[1:n]
	command := buf[0]
	p.Log = p.Log.WithField("command", cmdMap[command])
	// command only support connect
	if command != CmdConnect {
		logrus.Errorln(ErrRequestUnknownCode)
		return "", ErrRequestUnknownCode
	}
	user := p.readUntilNull(buf[7:])
	if p.Auth.Enable() && !p.Auth.Verify(user, "", p.srcAddr()) {
		_, _ = p.Src.Write([]byte{0x01, 0x00})
		p.Log.Errorln(ErrRequestIdentdMismatched)
		return "", ErrRequestIdentdMismatched

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

func (p *Proxy) processRequest(target string) {
	var err error
	p.Log = p.Log.WithField("dest", target)
	p.Log.Info("establish connection")
	// connect to the target
	p.Dest, err = p.Dial("tcp", target)
	if err != nil {
		// connection failed
		_, _ = p.Src.Write([]byte{0x00, 0x5b, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00})
		p.Log.Errorln(ErrRequestIdentdFailed, err)
		return
	}
	p.Log = p.Log.WithField("proxy", p.proxyAddr())
	p.Log = p.Log.WithField("dest", p.destAddr())
	_, err = p.Src.Write([]byte{0x00, 0x5A, 0x00, 0x00, 0, 0, 0, 0})
	if err != nil {
		p.Log.Errorln("write  response error", err)
		return
	}

	p.Log.Infoln("connection established")
	common.Forward(p.Src, p.Dest)
}

func (p *Proxy) readUntilNull(src []byte) string {
	buf := &bytes.Buffer{}
	for _, v := range src {
		if v == 0 {
			break
		}
		buf.WriteByte(v)
	}
	return buf.String()
}
