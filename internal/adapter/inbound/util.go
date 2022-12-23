package inbound

import (
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/transport/socks5"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func parseSocksAddr(target socks5.Addr) *constant.Metadata {
	metadata := &constant.Metadata{}

	switch target[0] {
	case socks5.AtypDomainName:
		// trim for FQDN
		metadata.Host = strings.TrimRight(string(target[2:2+target[1]]), ".")
		metadata.DstPort = strconv.Itoa((int(target[2+target[1]]) << 8) | int(target[2+target[1]+1]))
	case socks5.AtypIPv4:
		ip := net.IP(target[1 : 1+net.IPv4len])
		metadata.DstIP = ip
		metadata.DstPort = strconv.Itoa((int(target[1+net.IPv4len]) << 8) | int(target[1+net.IPv4len+1]))
	case socks5.AtypIPv6:
		ip := net.IP(target[1 : 1+net.IPv6len])
		metadata.DstIP = ip
		metadata.DstPort = strconv.Itoa((int(target[1+net.IPv6len]) << 8) | int(target[1+net.IPv6len+1]))
	}

	return metadata
}

func parseHTTPAddr(request *http.Request) *constant.Metadata {
	host := request.URL.Hostname()
	port := request.URL.Port()
	if port == "" {
		port = "80"
	}

	// trim FQDN (#737)
	host = strings.TrimRight(host, ".")

	metadata := &constant.Metadata{
		NetWork: constant.TCP,
		Host:    host,
		DstIP:   nil,
		DstPort: port,
	}

	if ip := net.ParseIP(host); ip != nil {
		metadata.DstIP = ip
	}

	return metadata
}

func parseAddr(addr string) (net.IP, string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, "", err
	}

	ip := net.ParseIP(host)
	return ip, port, nil
}
