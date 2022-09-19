package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
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

func (p *Proxy) Handle(buf []byte, _ int) {
	p.Log = p.Log.WithField("src", p.srcAddr())
	lines, err := p.readString(buf, "\r\n")
	if err != nil {
		p.Log.Errorln(err)
		return
	}
	if len(lines) < 2 {
		p.Log.Errorln("request line error")
		return
	}
	err = p.handshake(lines)
	if err != nil {
		return
	}
	p.processRequest(lines)
	return
}

func (p *Proxy) handshake(lines []string) (err error) {
	var user, pass string
	for _, line := range lines {
		// get username/password
		if strings.HasPrefix(line, ProxyAuthorization) {
			line = strings.TrimPrefix(line, ProxyAuthorization)
			bs, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				p.Log.Errorln(err)
				continue
			}
			if bs == nil {
				continue
			}
			_auth := bytes.Split(bs, []byte(":"))
			if len(_auth) < 2 {
				continue
			}
			user, pass = string(_auth[0]), string(bytes.Join(_auth[1:], []byte(":")))
		}
	}
	if user != "" {
		p.Log = p.Log.WithField("user", user)
	}
	// check username/password
	if p.Auth.Enable() && !p.Auth.Verify(user, pass, p.srcAddr()) {
		_, err = p.Src.Write([]byte{0x00, 0xff})
		if err != nil {
			p.Log.Errorln(err)
			return err
		}
		p.Log.Errorln("authentication failed")
		return err
	}
	return nil
}

func (p *Proxy) processRequest(lines []string) {
	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 3 {
		p.Log.Errorln("request line error")
		return
	}
	method := requestLine[0]
	requestTarget := requestLine[1]
	version := requestLine[2]
	p.Log = p.Log.WithField("command", method)
	if method == HTTPCONNECT {
		shp := strings.Split(requestTarget, ":")
		addr := shp[0]
		port, _ := strconv.Atoi(shp[1])
		p.handleHTTPConnectMethod(addr, uint16(port))
		return
	} else {
		si := strings.Index(requestTarget, "//")
		restUrl := requestTarget[si+2:]
		if restUrl == "" {
			_, _ = p.Src.Write([]byte("HTTP/1.0 404 Not Found\r\n\r\n"))
			p.Log.Errorln("404 Not Found")
			return
		}
		port := 80
		ei := strings.Index(restUrl, "/")
		url := "/"
		hostPort := restUrl
		if ei != -1 {
			hostPort = restUrl[:ei]
			url = restUrl[ei:]
		}
		as := strings.Split(hostPort, ":")
		addr := as[0]
		if len(as) == 2 {
			port, _ = strconv.Atoi(as[1])
		}
		var header string
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, ProxyAuthorization) {
				continue
			}
			if strings.HasPrefix(line, "Proxy-") {
				line = strings.TrimPrefix(line, "Proxy-")
			}
			header += fmt.Sprintf("%s\r\n", line)
		}
		newline := method + " " + url + " " + version + "\r\n" + header
		p.handleHTTPProxy(addr, uint16(port), newline)
		return
	}
}

func (p *Proxy) httpWriteProxyHeader() {
	_, err := p.Src.Write([]byte("HTTP/1.1 200 OK Connection Established\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}

	_, err = p.Src.Write([]byte(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123))))
	if err != nil {
		logrus.Warningln(err)
		return
	}
	_, err = p.Src.Write([]byte("Transfer-Encoding: chunked\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}
	_, err = p.Src.Write([]byte("\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}
}

func (p *Proxy) handleHTTPConnectMethod(addr string, port uint16) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.Log = p.Log.WithField("target", target)
	p.Log.Info("establish connection")
	p.Dest, err = p.Dial("tcp", target)
	if err != nil {
		_, _ = p.Src.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		p.Log.Errorln("connect dist error", err)
		return
	}

	p.Log = p.Log.WithField("proxy", p.proxyAddr())
	p.Log = p.Log.WithField("dest", p.destAddr())
	p.httpWriteProxyHeader()
	p.Log.Infoln("connection established")
	common.Forward(p.Src, p.Dest)
	return
}

// Subsequent request lines are full paths, some servers may have problems
func (p *Proxy) handleHTTPProxy(addr string, port uint16, line string) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.Log = p.Log.WithField("dest", target)
	p.Log.Info("establish connection")
	p.Dest, err = p.Dial("tcp", target)
	if err != nil {
		p.Log.Errorln("connect dist error", err)
		return
	}

	p.Log = p.Log.WithField("proxy", p.proxyAddr())
	p.Log = p.Log.WithField("dest", p.destAddr())
	_, err = p.Dest.Write([]byte(line))
	if err != nil {
		p.Log.Errorln("write  response error", err)
		return
	}
	p.Log.Infoln("connection established")
	common.Forward(p.Src, p.Dest)
	return
}

func (p *Proxy) readString(buf []byte, delim string) ([]string, error) {
	_, err := io.ReadAtLeast(p.Src, buf, 0)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return strings.Split(string(buf), delim), nil
}
