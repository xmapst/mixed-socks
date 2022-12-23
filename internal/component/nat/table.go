package nat

import (
	"github.com/xmapst/mixed-socks/internal/constant"
	"sync"
)

type Table struct {
	mapping sync.Map
}

func (t *Table) Set(key string, pc constant.PacketConn) {
	t.mapping.Store(key, pc)
}

func (t *Table) Get(key string) constant.PacketConn {
	item, exist := t.mapping.Load(key)
	if !exist {
		return nil
	}
	return item.(constant.PacketConn)
}

func (t *Table) GetOrCreateLock(key string) (*sync.Cond, bool) {
	item, loaded := t.mapping.LoadOrStore(key, sync.NewCond(&sync.Mutex{}))
	return item.(*sync.Cond), loaded
}

func (t *Table) Delete(key string) {
	t.mapping.Delete(key)
}

// New return *Cache
func New() *Table {
	return &Table{}
}
