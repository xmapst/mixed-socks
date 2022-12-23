package dns

import (
	"github.com/xmapst/mixed-socks/internal/common/cache"
	"net"
)

type ResolverEnhancer struct {
	mapping *cache.LruCache
}

func (h *ResolverEnhancer) MappingEnabled() bool {
	return true
}
func (h *ResolverEnhancer) FindHostByIP(ip net.IP) (string, bool) {
	if mapping := h.mapping; mapping != nil {
		if host, existed := h.mapping.Get(ip.String()); existed {
			return host.(string), true
		}
	}
	return "", false
}

func (h *ResolverEnhancer) PatchFrom(o *ResolverEnhancer) {
	if h.mapping != nil && o.mapping != nil {
		o.mapping.CloneTo(h.mapping)
	}
}

func NewEnhancer() *ResolverEnhancer {
	return &ResolverEnhancer{
		mapping: cache.New(cache.WithSize(65535), cache.WithStale(true)),
	}
}
