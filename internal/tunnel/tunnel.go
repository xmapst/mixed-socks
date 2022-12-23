package tunnel

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/adapter"
	"github.com/xmapst/mixed-socks/internal/adapter/inbound"
	"github.com/xmapst/mixed-socks/internal/adapter/outbound"
	"github.com/xmapst/mixed-socks/internal/component/nat"
	"github.com/xmapst/mixed-socks/internal/component/resolver"
	"github.com/xmapst/mixed-socks/internal/constant"
	icontext "github.com/xmapst/mixed-socks/internal/context"
	"github.com/xmapst/mixed-socks/internal/tunnel/statistic"
	"net"
	"net/netip"
	"runtime"
	"time"
)

var (
	tcpQueue = make(chan constant.ConnContext, 65535)
	udpQueue = make(chan *inbound.PacketAdapter, 65535)
	natTable = nat.New()
	proxies  = adapter.NewProxy(outbound.NewDirect())

	// default timeout for UDP session
	udpTimeout = 60 * time.Second
	workers    = 4
)

func init() {
	go process()
}

// TCPIn return fan-in queue
func TCPIn() chan<- constant.ConnContext {
	return tcpQueue
}

// UDPIn return fan-in udp queue
func UDPIn() chan<- *inbound.PacketAdapter {
	return udpQueue
}

// processUDP starts a loop to handle udp packet
func processUDP() {
	queue := udpQueue
	for conn := range queue {
		handleUDPConn(conn)
	}
}

// processTCP starts a loop to handle udp packet
func processTCP() {
	queue := tcpQueue
	for conn := range queue {
		go handleTCPConn(conn)
	}
}

func process() {
	if num := runtime.GOMAXPROCS(0); num > workers {
		workers = num
	}
	workers *= workers
	for i := 0; i < workers; i++ {
		go processUDP()
		go processTCP()
	}
}

func needLookupIP(metadata *constant.Metadata) bool {
	return resolver.MappingEnabled() && metadata.Host == "" && metadata.DstIP != nil
}

func preHandleMetadata(metadata *constant.Metadata) error {
	// handle IP string on host
	if ip := net.ParseIP(metadata.Host); ip != nil {
		metadata.DstIP = ip
		metadata.Host = ""
	}

	// preprocess enhanced-mode metadata
	if needLookupIP(metadata) {
		host, exist := resolver.FindHostByIP(metadata.DstIP)
		if exist {
			metadata.Host = host
			if node := resolver.DefaultHosts.Search(host); node != nil {
				// redir-host should lookup the hosts
				metadata.DstIP = node.Data.(net.IP)
			}
		}
	}

	return nil
}

func handleUDPConn(packet *inbound.PacketAdapter) {
	metadata := packet.Metadata()
	if !metadata.Valid() {
		logrus.Warnf("[Metadata] not valid: %#v", metadata)
		return
	}
	if err := preHandleMetadata(metadata); err != nil {
		logrus.Debugf("[Metadata PreHandle] error: %s", err)
		return
	}

	// local resolve UDP dns
	if !metadata.Resolved() {
		ips, err := resolver.LookupIP(context.Background(), metadata.Host)
		if err != nil {
			return
		} else if len(ips) == 0 {
			return
		}
		metadata.DstIP = ips[0]
	}

	key := packet.LocalAddr().String()

	handle := func() bool {
		pc := natTable.Get(key)
		if pc != nil {
			_ = handleUDPToRemote(packet, pc, metadata)
			return true
		}
		return false
	}

	if handle() {
		return
	}

	lockKey := key + "-lock"
	cond, loaded := natTable.GetOrCreateLock(lockKey)

	go func() {
		if loaded {
			cond.L.Lock()
			cond.Wait()
			handle()
			cond.L.Unlock()
			return
		}

		defer func() {
			natTable.Delete(lockKey)
			cond.Broadcast()
		}()

		pCtx := icontext.NewPacketConnContext(metadata)
		ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultUDPTimeout)
		defer cancel()
		rawPc, err := proxies.ListenPacketContext(ctx, metadata.Pure())
		if err != nil {
			logrus.Warnf("[UDP] dial %s --> %s error: %s", metadata.SourceAddress(), metadata.RemoteAddress(), err.Error())
			return
		}

		pCtx.InjectPacketConn(rawPc)
		pc := statistic.NewUDPTracker(rawPc, statistic.DefaultManager, metadata)
		logrus.Infof("[UDP] %s --> %s", metadata.SourceAddress(), metadata.RemoteAddress())

		oAddr, _ := netip.AddrFromSlice(metadata.DstIP)
		oAddr = oAddr.Unmap()
		go handleUDPToLocal(packet.UDPPacket, pc, key, oAddr)

		natTable.Set(key, pc)
		handle()
	}()
}

func handleTCPConn(connCtx constant.ConnContext) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(connCtx.Conn())

	metadata := connCtx.Metadata()
	if !metadata.Valid() {
		logrus.Warnf("[Metadata] not valid: %#v", metadata)
		return
	}

	if err := preHandleMetadata(metadata); err != nil {
		logrus.Debugf("[Metadata PreHandle] error: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultTCPTimeout)
	defer cancel()
	remoteConn, err := proxies.DialContext(ctx, metadata.Pure())
	if err != nil {
		logrus.Warnf("[%s] %s --> %s error: %s", metadata.Type.String(), metadata.SourceAddress(), metadata.RemoteAddress(), err.Error())
		return
	}
	remoteConn = statistic.NewTCPTracker(remoteConn, statistic.DefaultManager, metadata)
	defer func(remoteConn constant.Conn) {
		_ = remoteConn.Close()
	}(remoteConn)

	logrus.Infof("[%s] %s --> %s", metadata.Type.String(), metadata.SourceAddress(), metadata.RemoteAddress())
	handleSocket(connCtx, remoteConn)
}
