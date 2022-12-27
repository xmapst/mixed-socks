package dns

import (
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/common/cache"
	"github.com/xmapst/mixed-socks/internal/component/trie"
	"github.com/xmapst/mixed-socks/internal/context"
	"net"
	"strings"
	"time"
)

type (
	handler    func(ctx *context.DNSContext, r *dns.Msg) (*dns.Msg, error)
	middleware func(next handler) handler
)

func withHosts(hosts *trie.DomainTrie) middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, r *dns.Msg) (*dns.Msg, error) {
			ctx.SetType(context.DNSTypeHost)
			q := r.Question[0]

			if !isIPRequest(q) {
				return next(ctx, r)
			}

			record := hosts.Search(strings.TrimRight(q.Name, "."))
			if record == nil {
				return next(ctx, r)
			}

			ip := record.Data.(net.IP)
			msg := r.Copy()

			if v4 := ip.To4(); v4 != nil && q.Qtype == dns.TypeA {
				rr := &dns.A{}
				rr.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: dnsDefaultTTL}
				rr.A = v4

				msg.Answer = []dns.RR{rr}
			} else if v6 := ip.To16(); v6 != nil && q.Qtype == dns.TypeAAAA {
				rr := &dns.AAAA{}
				rr.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: dnsDefaultTTL}
				rr.AAAA = v6

				msg.Answer = []dns.RR{rr}
			} else {
				return next(ctx, r)
			}
			logrus.Infof("[DNS] %s --> %s --> %s", ctx.RemoteAddr().String(), strings.TrimSuffix(q.Name, "."), ip)
			msg.SetRcode(r, dns.RcodeSuccess)
			msg.Authoritative = true
			msg.RecursionAvailable = true

			return msg, nil
		}
	}
}

func withMapping(mapping *cache.LruCache) middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, r *dns.Msg) (*dns.Msg, error) {
			ctx.SetType(context.DNSTypeCache)
			q := r.Question[0]

			if !isIPRequest(q) {
				return next(ctx, r)
			}

			msg, err := next(ctx, r)
			if err != nil {
				return nil, err
			}

			host := strings.TrimRight(q.Name, ".")

			for _, ans := range msg.Answer {
				var ip net.IP
				var ttl uint32

				switch a := ans.(type) {
				case *dns.A:
					ip = a.A
					ttl = a.Hdr.Ttl
				case *dns.AAAA:
					ip = a.AAAA
					ttl = a.Hdr.Ttl
				default:
					continue
				}
				logrus.Infof("[DNS] %s --> %s --> %s", ctx.RemoteAddr().String(), strings.TrimSuffix(q.Name, "."), ip)
				mapping.SetWithExpire(ip.String(), host, time.Now().Add(time.Second*time.Duration(ttl)))
				msg.SetRcode(r, dns.RcodeSuccess)
			}

			return msg, nil
		}
	}
}

func withResolver(resolver *Resolver) handler {
	return func(ctx *context.DNSContext, r *dns.Msg) (*dns.Msg, error) {
		ctx.SetType(context.DNSTypeRaw)
		q := r.Question[0]

		// return a empty AAAA msg when ipv6 disabled
		if q.Qtype == dns.TypeAAAA {
			return handleMsgWithEmptyAnswer(r), nil
		}

		msg, err := resolver.Exchange(r)
		if err != nil {
			logrus.Debugln("[DNS] exchange --> %s failed: %v", q.String(), err)
			return msg, err
		}
		msg.SetRcode(r, msg.Rcode)
		msg.Authoritative = true

		return msg, nil
	}
}

func compose(middlewares []middleware, endpoint handler) handler {
	length := len(middlewares)
	h := endpoint
	for i := length - 1; i >= 0; i-- {
		m := middlewares[i]
		h = m(h)
	}

	return h
}

func newHandler(resolver *Resolver, mapper *ResolverEnhancer) handler {
	var middlewares []middleware

	if resolver.hosts != nil {
		middlewares = append(middlewares, withHosts(resolver.hosts))
	}

	middlewares = append(middlewares, withMapping(mapper.mapping))

	return compose(middlewares, withResolver(resolver))
}
