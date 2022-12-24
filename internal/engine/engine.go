package engine

import (
	"github.com/sirupsen/logrus"
	N "github.com/xmapst/mixed-socks/internal/common/net"
	"github.com/xmapst/mixed-socks/internal/component/auth"
	"github.com/xmapst/mixed-socks/internal/component/dialer"
	"github.com/xmapst/mixed-socks/internal/component/iface"
	"github.com/xmapst/mixed-socks/internal/component/resolver"
	"github.com/xmapst/mixed-socks/internal/component/trie"
	"github.com/xmapst/mixed-socks/internal/config"
	"github.com/xmapst/mixed-socks/internal/controller"
	"github.com/xmapst/mixed-socks/internal/dns"
	"github.com/xmapst/mixed-socks/internal/listener"
	authStore "github.com/xmapst/mixed-socks/internal/listener/auth"
	"github.com/xmapst/mixed-socks/internal/tunnel"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"net"
	"os"
	"path/filepath"
)

// Run call at the beginning of mixed-socks
func Run() error {
	var changeCh = make(chan bool, 1024)
	go applyConfig(changeCh)

	err := config.Load(changeCh)
	if err != nil {
		return err
	}
	cfg := config.App.Controller
	if cfg.Enable && cfg.Port != 0 {
		addr := N.GenAddr(cfg.Listen, cfg.Port)
		go controller.Start(addr, cfg.Secret)
	}
	return nil
}

// applyConfig dispatch configure to all parts
func applyConfig(changeCh chan bool) {
	for range changeCh {
		updateOutbound(config.App.Outbound)
		updateLogger(config.App.Log)
		updateWhitelist(config.App.Whitelist)
		updateUsers(config.App.Users)
		updateHosts(config.App.Hosts)
		updateInbound(config.App.Inbound)
		updateDNS(config.App.DNS)
	}
}

var _fileName string

func updateLogger(cfg *config.Log) {
	if cfg == nil {
		logrus.SetOutput(io.Discard)
		if cfg.Output != nil {
			err := cfg.Output.Close()
			if err != nil {
				logrus.Warnln(err)
			}
		}
		return
	}
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	if cfg.Filename == _fileName {
		return
	}
	if cfg.Filename != "" && cfg.Filename != "stdout" {
		err = os.MkdirAll(filepath.Dir(cfg.Filename), 0777)
		if err != nil {
			logrus.Errorln(err)
			return
		}
		cfg.Output = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxBackups: cfg.MaxBackups,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}
		logrus.SetOutput(cfg.Output)
	} else {
		cfg.Output = nil
		logrus.SetOutput(os.Stdout)
	}
	_fileName = cfg.Filename
}

func updateDNS(c *config.DNS) {
	if !c.Enable {
		resolver.DefaultResolver = nil
		resolver.DefaultHostMapper = nil
		dns.ReCreateServer("", nil, nil)
		return
	}

	cfg := dns.Config{
		NameServers: c.NameServers,
		Hosts:       c.Hosts,
	}

	r := dns.NewResolver(cfg)
	m := dns.NewEnhancer()

	// reuse cache of old host mapper
	if old := resolver.DefaultHostMapper; old != nil {
		m.PatchFrom(old.(*dns.ResolverEnhancer))
	}

	resolver.DefaultResolver = r
	resolver.DefaultHostMapper = m
	addr := N.GenAddr(c.Listen, c.Port)
	dns.ReCreateServer(addr, r, m)
}

func updateHosts(tree *trie.DomainTrie) {
	resolver.DefaultHosts = tree
}

func updateOutbound(cfg *config.Outbound) {
	if cfg.Interface != "" {
		iface, err := net.InterfaceByName(cfg.Interface)
		if err == nil {
			dialer.WithInterface(iface.Name)
			logrus.Infof("dialer bind to interface: %s", cfg.Interface)
		}
	}
	if cfg.RoutingMark != 0 {
		dialer.WithRoutingMark(cfg.RoutingMark)
		logrus.Infof("dialer set fwmark: %#x", cfg.RoutingMark)
	}
}

func updateInbound(cfg *config.Inbound) {
	iface.FlushCache()

	tcpIn := tunnel.TCPIn()
	udpIn := tunnel.UDPIn()
	addr := N.GenAddr(cfg.Listen, cfg.Port)
	listener.ReCreateMixed(addr, tcpIn, udpIn)
}

func updateUsers(users []auth.AuthUser) {
	authenticator := auth.NewAuthenticator(users)
	authStore.SetAuthenticator(authenticator)
	if authenticator != nil {
		logrus.Infoln("Authentication of local server updated")
	}
}

func updateWhitelist(ips []net.IP) {
	authenticator := auth.NewWhitelist(ips)
	authStore.SetWhitelist(authenticator)
	if authenticator != nil {
		logrus.Infoln("Whitelist of local server updated")
	}
}
