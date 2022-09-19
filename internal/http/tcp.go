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

type proxy struct {
	log  *logrus.Entry
	src  net.Conn
	dest net.Conn
	auth auth.Authenticator
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

func Handle(src net.Conn, buf []byte, auth auth.Authenticator) net.Conn {
	d := net.Dialer{Timeout: 10 * time.Second}
	p := &proxy{
		src:  src,
		auth: auth,
		log: logrus.WithFields(logrus.Fields{
			"uuid": common.GUID(),
		}),
		dial: d.Dial,
	}
	p.log = p.log.WithField("src", p.srcAddr())
	lines, err := p.readString(buf, "\r\n")
	if err != nil {
		p.log.Errorln(err)
		return p.dest
	}
	if len(lines) < 2 {
		p.log.Errorln("request line error")
		return p.dest
	}
	err = p.handshake(lines)
	if err != nil {
		return p.dest
	}
	p.processRequest(lines)
	return p.dest
}

func (p *proxy) handshake(lines []string) (err error) {
	var user, pass string
	for _, line := range lines {
		// get username/password
		if strings.HasPrefix(line, ProxyAuthorization) {
			line = strings.TrimPrefix(line, ProxyAuthorization)
			bs, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				p.log.Errorln(err)
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
		p.log = p.log.WithField("user", user)
	}
	// check username/password
	if p.auth.Enable() && !p.auth.Verify(user, pass, p.srcAddr()) {
		_, err = p.src.Write([]byte{0x00, 0xff})
		if err != nil {
			p.log.Errorln(err)
			return err
		}
		p.log.Errorln("authentication failed")
		return err
	}
	return nil
}

func (p *proxy) processRequest(lines []string) {
	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 3 {
		p.log.Errorln("request line error")
		return
	}
	method := requestLine[0]
	requestTarget := requestLine[1]
	version := requestLine[2]
	p.log = p.log.WithField("command", method)
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
			_, _ = p.src.Write([]byte("HTTP/1.0 404 Not Found\r\n\r\n"))
			p.log.Errorln("404 Not Found")
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

func (p *proxy) httpWriteProxyHeader() {
	_, err := p.src.Write([]byte("HTTP/1.1 200 OK Connection Established\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}

	_, err = p.src.Write([]byte(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123))))
	if err != nil {
		logrus.Warningln(err)
		return
	}
	_, err = p.src.Write([]byte("Transfer-Encoding: chunked\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}
	_, err = p.src.Write([]byte("\r\n"))
	if err != nil {
		logrus.Warningln(err)
		return
	}
}

func (p *proxy) handleHTTPConnectMethod(addr string, port uint16) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.log = p.log.WithField("target", target)
	p.dest, err = p.dial("tcp", target)
	if err != nil {
		_, _ = p.src.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		p.log.Errorln("connect dist error", err)
		return
	}

	p.log = p.log.WithField("proxy", p.proxyAddr())
	p.log = p.log.WithField("dest", p.destAddr())
	p.httpWriteProxyHeader()
	p.log.Infoln("connection established")
	common.Forward(p.src, p.dest)
	return
}

// Subsequent request lines are full paths, some servers may have problems
func (p *proxy) handleHTTPProxy(addr string, port uint16, line string) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.log = p.log.WithField("target", target)
	p.dest, err = p.dial("tcp", target)
	if err != nil {
		p.log.Errorln("connect dist error", err)
		return
	}

	p.log = p.log.WithField("proxy", p.proxyAddr())
	p.log = p.log.WithField("dest", p.destAddr())
	_, err = p.dest.Write([]byte(line))
	if err != nil {
		p.log.Errorln("write  response error", err)
		return
	}
	p.log.Infoln("connection established")
	common.Forward(p.src, p.dest)
	return
}

func (p *proxy) readString(buf []byte, delim string) ([]string, error) {
	_, err := io.ReadAtLeast(p.src, buf, 0)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return strings.Split(string(buf), delim), nil
}