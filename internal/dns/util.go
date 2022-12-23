package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/common/cache"
	"github.com/xmapst/mixed-socks/internal/common/picker"
	"net"
	"time"
)

func putMsgToCache(c *cache.LruCache, key string, msg *dns.Msg) {
	var ttl uint32
	switch {
	case len(msg.Answer) != 0:
		ttl = msg.Answer[0].Header().Ttl
	case len(msg.Ns) != 0:
		ttl = msg.Ns[0].Header().Ttl
	case len(msg.Extra) != 0:
		ttl = msg.Extra[0].Header().Ttl
	default:
		logrus.Debugln("[DNS] response msg empty: %#v", msg)
		return
	}

	c.SetWithExpire(key, msg.Copy(), time.Now().Add(time.Second*time.Duration(ttl)))
}

func setMsgTTL(msg *dns.Msg, ttl uint32) {
	for _, answer := range msg.Answer {
		answer.Header().Ttl = ttl
	}

	for _, ns := range msg.Ns {
		ns.Header().Ttl = ttl
	}

	for _, extra := range msg.Extra {
		extra.Header().Ttl = ttl
	}
}

func isIPRequest(q dns.Question) bool {
	return q.Qclass == dns.ClassINET && (q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA)
}

func transform(servers []NameServer, resolver *Resolver) []dnsClient {
	var ret []dnsClient
	for _, s := range servers {
		switch s.Net {
		case "https":
			ret = append(ret, newDoHClient(s.Addr, s.Interface, resolver))
			continue
		case "dhcp":
			ret = append(ret, newDHCPClient(s.Addr))
			continue
		}

		host, port, _ := net.SplitHostPort(s.Addr)
		ret = append(ret, &client{
			Client: &dns.Client{
				Net: s.Net,
				TLSConfig: &tls.Config{
					ServerName: host,
				},
				UDPSize: 4096,
				Timeout: 5 * time.Second,
			},
			port:  port,
			host:  host,
			iface: s.Interface,
			r:     resolver,
		})
	}
	return ret
}

func handleMsgWithEmptyAnswer(r *dns.Msg) *dns.Msg {
	msg := &dns.Msg{}
	msg.Answer = []dns.RR{}

	msg.SetRcode(r, dns.RcodeSuccess)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	return msg
}

func msgToIP(msg *dns.Msg) []net.IP {
	var ips []net.IP

	for _, answer := range msg.Answer {
		switch ans := answer.(type) {
		case *dns.AAAA:
			ips = append(ips, ans.AAAA)
		case *dns.A:
			ips = append(ips, ans.A)
		}
	}

	return ips
}

func batchExchange(ctx context.Context, clients []dnsClient, m *dns.Msg) (msg *dns.Msg, err error) {
	fast, ctx := picker.WithContext(ctx)
	for _, client := range clients {
		r := client
		fast.Go(func() (any, error) {
			m, err := r.ExchangeContext(ctx, m)
			if err != nil {
				return nil, err
			} else if m.Rcode == dns.RcodeServerFailure || m.Rcode == dns.RcodeRefused {
				return nil, errors.New("server failure")
			}
			return m, nil
		})
	}

	elm := fast.Wait()
	if elm == nil {
		err := errors.New("all DNS requests failed")
		if fErr := fast.Error(); fErr != nil {
			err = fmt.Errorf("%w, first error: %s", err, fErr.Error())
		}
		return nil, err
	}

	msg = elm.(*dns.Msg)
	return
}
