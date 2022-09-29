package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/auth"
	"github/xmapst/mixed-socks/internal/common"
	"github/xmapst/mixed-socks/internal/statistic"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Proxy struct {
	uuid string
	log  *logrus.Entry
	src  net.Conn
	dest net.Conn
	auth auth.Authenticator
	dial common.DialFunc
}

func (p *Proxy) srcAddr() string {
	if p.src != nil {
		return p.src.RemoteAddr().String()
	}
	return ""
}

func (p *Proxy) proxyAddr() string {
	if p.dest != nil {
		return p.dest.LocalAddr().String()
	}
	return ""
}

func (p *Proxy) destAddr() string {
	if p.dest != nil {
		return p.dest.RemoteAddr().String()
	}
	return ""
}

func (p *Proxy) init(uuid string, conn net.Conn, authenticator auth.Authenticator, dial common.DialFunc, log *logrus.Entry) {
	p.uuid = uuid
	p.src = conn
	p.auth = authenticator
	p.dial = dial
	p.log = log
}

func (p *Proxy) Handle(uuid string, conn net.Conn, authenticator auth.Authenticator, dial common.DialFunc, log *logrus.Entry) {
	p.init(uuid, conn, authenticator, dial, log)
	p.log = p.log.WithField("src", p.srcAddr())
	lines, err := p.readString("\r\n")
	if err != nil {
		p.log.Errorln(err)
		return
	}
	if len(lines) < 2 {
		p.log.Errorln("request line error")
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

func (p *Proxy) processRequest(lines []string) {
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

func (p *Proxy) httpWriteProxyHeader() {
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

func (p *Proxy) handleHTTPConnectMethod(addr string, port uint16) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.log = p.log.WithField("target", target)
	p.log.Info("establish connection")
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
	srcIP, srcPort, _ := net.SplitHostPort(p.srcAddr())
	destIP, destPort, _ := net.SplitHostPort(p.destAddr())
	common.Forward(p.uuid, p.src, p.dest, &statistic.Metadata{
		NetWork:  statistic.TCP,
		Type:     statistic.HTTPCONNECT,
		SrcIP:    srcIP,
		SrcPort:  srcPort,
		DestIP:   destIP,
		DestPort: destPort,
		Host:     target,
	})
	return
}

// Subsequent request lines are full paths, some servers may have problems
func (p *Proxy) handleHTTPProxy(addr string, port uint16, line string) {
	var err error
	target := fmt.Sprintf("%s:%d", addr, port)
	p.log = p.log.WithField("dest", target)
	p.log.Info("establish connection")
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
	srcIP, srcPort, _ := net.SplitHostPort(p.srcAddr())
	destIP, destPort, _ := net.SplitHostPort(p.destAddr())
	common.Forward(p.uuid, p.src, p.dest, &statistic.Metadata{
		NetWork:  statistic.TCP,
		Type:     statistic.HTTP,
		SrcIP:    srcIP,
		SrcPort:  srcPort,
		DestIP:   destIP,
		DestPort: destPort,
		Host:     target,
	})
	return
}

func (p *Proxy) readString(delim string) ([]string, error) {
	var buf = make([]byte, 4096)
	_, err := io.ReadAtLeast(p.src, buf, 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return strings.Split(string(buf), delim), nil
}
