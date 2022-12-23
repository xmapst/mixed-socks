package dns

import (
	"context"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/common/cache"
	"github.com/xmapst/mixed-socks/internal/component/resolver"
	"github.com/xmapst/mixed-socks/internal/component/trie"
	"golang.org/x/sync/singleflight"
	"math/rand"
	"net"
	"strings"
	"time"
)

type dnsClient interface {
	Exchange(m *dns.Msg) (msg *dns.Msg, err error)
	ExchangeContext(ctx context.Context, m *dns.Msg) (msg *dns.Msg, err error)
}

type result struct {
	Msg   *dns.Msg
	Error error
}

type Resolver struct {
	hosts    *trie.DomainTrie
	main     []dnsClient
	group    singleflight.Group
	lruCache *cache.LruCache
}

// LookupIP request with TypeA and TypeAAAA, priority return TypeA
func (r *Resolver) LookupIP(ctx context.Context, host string) (ip []net.IP, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan []net.IP, 1)

	go func() {
		defer close(ch)
		ip, err := r.lookupIP(ctx, host, dns.TypeAAAA)
		if err != nil {
			return
		}
		ch <- ip
	}()

	ip, err = r.lookupIP(ctx, host, dns.TypeA)
	if err == nil {
		return
	}

	ip, open := <-ch
	if !open {
		return nil, resolver.ErrIPNotFound
	}

	return ip, nil
}

// ResolveIP request with TypeA and TypeAAAA, priority return TypeA
func (r *Resolver) ResolveIP(host string) (ip net.IP, err error) {
	ips, err := r.LookupIP(context.Background(), host)
	if err != nil {
		return nil, err
	} else if len(ips) == 0 {
		return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, host)
	}
	return ips[rand.Intn(len(ips))], nil
}

// LookupIPv4 request with TypeA
func (r *Resolver) LookupIPv4(ctx context.Context, host string) ([]net.IP, error) {
	return r.lookupIP(ctx, host, dns.TypeA)
}

// ResolveIPv4 request with TypeA
func (r *Resolver) ResolveIPv4(host string) (ip net.IP, err error) {
	ips, err := r.lookupIP(context.Background(), host, dns.TypeA)
	if err != nil {
		return nil, err
	} else if len(ips) == 0 {
		return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, host)
	}
	return ips[rand.Intn(len(ips))], nil
}

// LookupIPv6 request with TypeAAAA
func (r *Resolver) LookupIPv6(ctx context.Context, host string) ([]net.IP, error) {
	return r.lookupIP(ctx, host, dns.TypeAAAA)
}

// ResolveIPv6 request with TypeAAAA
func (r *Resolver) ResolveIPv6(host string) (ip net.IP, err error) {
	ips, err := r.lookupIP(context.Background(), host, dns.TypeAAAA)
	if err != nil {
		return nil, err
	} else if len(ips) == 0 {
		return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, host)
	}
	return ips[rand.Intn(len(ips))], nil
}

// Exchange a batch of dns request, and it use cache
func (r *Resolver) Exchange(m *dns.Msg) (msg *dns.Msg, err error) {
	return r.ExchangeContext(context.Background(), m)
}

// ExchangeContext a batch of dns request with context.Context, and it use cache
func (r *Resolver) ExchangeContext(ctx context.Context, m *dns.Msg) (msg *dns.Msg, err error) {
	if len(m.Question) == 0 {
		return nil, errors.New("should have one question at least")
	}

	q := m.Question[0]
	c, expireTime, hit := r.lruCache.GetWithExpire(q.String())
	if hit {
		now := time.Now()
		msg = c.(*dns.Msg).Copy()
		if expireTime.Before(now) {
			setMsgTTL(msg, uint32(1)) // Continue fetch
			go func() {
				_, err = r.exchangeWithoutCache(ctx, m)
				if err != nil {
					logrus.Warnln(err.Error())
				}
			}()
		} else {
			setMsgTTL(msg, uint32(time.Until(expireTime).Seconds()))
		}
		return
	}
	return r.exchangeWithoutCache(ctx, m)
}

// ExchangeWithoutCache a batch of dns request, and it do NOT GET from cache
func (r *Resolver) exchangeWithoutCache(ctx context.Context, m *dns.Msg) (msg *dns.Msg, err error) {
	q := m.Question[0]

	ret, err, shared := r.group.Do(q.String(), func() (result any, err error) {
		defer func() {
			if err != nil {
				return
			}

			msg := result.(*dns.Msg)

			putMsgToCache(r.lruCache, q.String(), msg)
		}()

		isIPReq := isIPRequest(q)
		if isIPReq {
			return r.ipExchange(ctx, m)
		}
		return r.batchExchange(ctx, r.main, m)
	})

	if err == nil {
		msg = ret.(*dns.Msg)
		if shared {
			msg = msg.Copy()
		}
	}

	return
}

func (r *Resolver) batchExchange(ctx context.Context, clients []dnsClient, m *dns.Msg) (msg *dns.Msg, err error) {
	ctx, cancel := context.WithTimeout(ctx, resolver.DefaultDNSTimeout)
	defer cancel()

	return batchExchange(ctx, clients, m)
}

func (r *Resolver) ipExchange(ctx context.Context, m *dns.Msg) (msg *dns.Msg, err error) {
	msgCh := r.asyncExchange(ctx, r.main, m)
	res := <-msgCh
	msg, err = res.Msg, res.Error
	return
}

func (r *Resolver) lookupIP(_ context.Context, host string, dnsType uint16) ([]net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		ip4 := ip.To4()
		isIPv4 := ip4 != nil
		if dnsType == dns.TypeAAAA && !isIPv4 {
			return []net.IP{ip}, nil
		} else if dnsType == dns.TypeA && isIPv4 {
			return []net.IP{ip4}, nil
		} else {
			return nil, resolver.ErrIPVersion
		}
	}

	query := &dns.Msg{}
	query.SetQuestion(dns.Fqdn(host), dnsType)

	msg, err := r.Exchange(query)
	if err != nil {
		return nil, err
	}

	ips := msgToIP(msg)
	if len(ips) == 0 {
		return nil, resolver.ErrIPNotFound
	}
	return ips, nil
}

func (r *Resolver) msgToDomain(msg *dns.Msg) string {
	if len(msg.Question) > 0 {
		return strings.TrimRight(msg.Question[0].Name, ".")
	}

	return ""
}

func (r *Resolver) asyncExchange(ctx context.Context, client []dnsClient, msg *dns.Msg) <-chan *result {
	ch := make(chan *result, 1)
	go func() {
		res, err := r.batchExchange(ctx, client, msg)
		ch <- &result{Msg: res, Error: err}
	}()
	return ch
}

type NameServer struct {
	Net       string
	Addr      string
	Interface string
}

type Config struct {
	NameServers []NameServer
	Hosts       *trie.DomainTrie
}

func NewResolver(config Config) *Resolver {
	r := &Resolver{
		main:     transform(config.NameServers, nil),
		lruCache: cache.New(cache.WithSize(4096), cache.WithStale(true)),
		hosts:    config.Hosts,
	}
	return r
}
