package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xmapst/mixed-socks/internal/common/queue"
	"github.com/xmapst/mixed-socks/internal/component/dialer"
	"github.com/xmapst/mixed-socks/internal/constant"
	"go.uber.org/atomic"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Proxy struct {
	constant.ProxyAdapter
	history *queue.Queue
	alive   *atomic.Bool
}

// Alive implements constant.Proxy
func (p *Proxy) Alive() bool {
	return p.alive.Load()
}

// Dial implements constant.Proxy
func (p *Proxy) Dial(metadata *constant.Metadata) (constant.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultTCPTimeout)
	defer cancel()
	return p.DialContext(ctx, metadata)
}

// DialContext implements constant.ProxyAdapter
func (p *Proxy) DialContext(ctx context.Context, metadata *constant.Metadata, opts ...dialer.Option) (constant.Conn, error) {
	conn, err := p.ProxyAdapter.DialContext(ctx, metadata, opts...)
	p.alive.Store(err == nil)
	return conn, err
}

// DialUDP implements constant.ProxyAdapter
func (p *Proxy) DialUDP(metadata *constant.Metadata) (constant.PacketConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultUDPTimeout)
	defer cancel()
	return p.ListenPacketContext(ctx, metadata)
}

// ListenPacketContext implements constant.ProxyAdapter
func (p *Proxy) ListenPacketContext(ctx context.Context, metadata *constant.Metadata, opts ...dialer.Option) (constant.PacketConn, error) {
	pc, err := p.ProxyAdapter.ListenPacketContext(ctx, metadata, opts...)
	p.alive.Store(err == nil)
	return pc, err
}

// DelayHistory implements constantProxy
func (p *Proxy) DelayHistory() []constant.DelayHistory {
	queues := p.history.Copy()
	var histories []constant.DelayHistory
	for _, item := range queues {
		histories = append(histories, item.(constant.DelayHistory))
	}
	return histories
}

// LastDelay return last history record. if proxy is not alive, return the max value of uint16.
// implements constant.Proxy
func (p *Proxy) LastDelay() (delay uint16) {
	var max uint16 = 0xffff
	if !p.alive.Load() {
		return max
	}

	last := p.history.Last()
	if last == nil {
		return max
	}
	history := last.(constant.DelayHistory)
	if history.Delay == 0 {
		return max
	}
	return history.Delay
}

// MarshalJSON implements constant.ProxyAdapter
func (p *Proxy) MarshalJSON() ([]byte, error) {
	inner, err := p.ProxyAdapter.MarshalJSON()
	if err != nil {
		return inner, err
	}

	mapping := map[string]any{}
	_ = json.Unmarshal(inner, &mapping)
	mapping["history"] = p.DelayHistory()
	mapping["name"] = p.Name()
	mapping["udp"] = p.SupportUDP()
	return json.Marshal(mapping)
}

// URLTest get the delay for the specified URL
// implements constant.Proxy
func (p *Proxy) URLTest(ctx context.Context, url string) (t uint16, err error) {
	defer func() {
		p.alive.Store(err == nil)
		record := constant.DelayHistory{Time: time.Now()}
		if err == nil {
			record.Delay = t
		}
		p.history.Put(record)
		if p.history.Len() > 10 {
			p.history.Pop()
		}
	}()

	addr, err := urlToMetadata(url)
	if err != nil {
		return
	}

	start := time.Now()
	instance, err := p.DialContext(ctx, &addr)
	if err != nil {
		return
	}
	defer func(instance constant.Conn) {
		_ = instance.Close()
	}(instance)

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	transport := &http.Transport{
		Dial: func(string, string) (net.Conn, error) {
			return instance, nil
		},
		// from http.DefaultTransport
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
	t = uint16(time.Since(start) / time.Millisecond)
	return
}

func NewProxy(adapter constant.ProxyAdapter) *Proxy {
	return &Proxy{adapter, queue.New(10), atomic.NewBool(true)}
}

func urlToMetadata(rawURL string) (addr constant.Metadata, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			err = fmt.Errorf("%s scheme not Support", rawURL)
			return
		}
	}

	addr = constant.Metadata{
		Host:    u.Hostname(),
		DstIP:   nil,
		DstPort: port,
	}
	return
}
