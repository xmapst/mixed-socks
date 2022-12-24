package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/xmapst/mixed-socks/internal/component/auth"
	"github.com/xmapst/mixed-socks/internal/component/iface"
	"github.com/xmapst/mixed-socks/internal/component/trie"
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/dns"
	"gopkg.in/natefinch/lumberjack.v2"
	"net"
	"net/url"
	"strings"
)

var (
	App = new(Config)
	v   = viper.NewWithOptions(viper.KeyDelimiter("::"))
)

type Config struct {
	Inbound    *Inbound
	Controller *Controller
	DNS        *DNS
	Hosts      *trie.DomainTrie
	Users      []auth.AuthUser
	Whitelist  []net.IP
	Log        *Log
}

// Inbound config
type Inbound struct {
	Listen      string `yaml:",default=0.0.0.0"`
	Port        int    `yaml:",default=8090"`
	Interface   string `yaml:""`
	RoutingMark int    `yaml:""`
}

type RawConfig struct {
	Inbound    *Inbound          `yaml:""`
	Controller *Controller       `yaml:""`
	Auth       map[string]string `yaml:""`
	Hosts      map[string]string `yaml:""`
	DNS        RawDNS            `yaml:""`
	Log        *Log              `yaml:""`
	WhiteList  []string          `yaml:""`
}

type Controller struct {
	Enable bool   `yaml:",default=false"`
	Listen string `yaml:",default=0.0.0.0"`
	Port   int    `yaml:",default=8080"`
	Secret string `yaml:""`
}

type RawDNS struct {
	Enable      bool     `yaml:",default=true"`
	NameServers []string `yaml:",default=8.8.8.8"`
	Listen      string   `yaml:",default=0.0.0.0"`
	Port        int      `yaml:",default=53"`
}

type DNS struct {
	Enable      bool             `yaml:""`
	NameServers []dns.NameServer `yaml:""`
	Listen      string           `yaml:""`
	Port        int              `yaml:""`
	Hosts       *trie.DomainTrie
}

type Log struct {
	Output     *lumberjack.Logger `json:"-" yaml:"-"`
	Filename   string             `yaml:""`
	Level      string             `yaml:",default=info"`
	MaxBackups int                `yaml:",default=7"`
	MaxSize    int                `yaml:",default=500"`
	MaxAge     int                `yaml:",default=28"`
	Compress   bool               `yaml:",default=true"`
}

func viperLoadConf() (*RawConfig, error) {
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var conf = &RawConfig{
		Inbound: &Inbound{
			Listen: "0.0.0.0",
			Port:   8090,
		},
		Controller: &Controller{
			Enable: false,
		},
		Hosts: map[string]string{},
		DNS: RawDNS{
			Enable: false,
			NameServers: []string{
				"114.114.114.114",
				"8.8.8.8",
			},
		},
		Log: &Log{
			Level:      "info",
			MaxBackups: 7,
			MaxSize:    500,
			MaxAge:     28,
			Compress:   true,
		},
	}
	err = v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func Load(changeCh chan bool) error {
	v.SetConfigFile(constant.Path.Config())
	v.SetConfigType("yaml")
	conf, err := viperLoadConf()
	if err != nil {
		return err
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		if !e.Has(fsnotify.Write) {
			return
		}
		logrus.Infoln(e.Name, "config file modified")
		conf, err := viperLoadConf()
		if err != nil {
			logrus.Errorln(err)
			return
		}
		err = conf.Parge()
		if err != nil {
			logrus.Errorln(err)
			return
		}
		changeCh <- true
	})
	err = conf.Parge()
	if err != nil {
		return err
	}
	c := cron.New()
	_, _ = c.AddFunc("@daily", func() {
		if App.Log.Output != nil {
			_ = App.Log.Output.Rotate()
		}
	})
	c.Start()
	changeCh <- true
	return nil
}

func (c *RawConfig) Parge() error {
	App = &Config{
		Inbound:    c.Inbound,
		Log:        c.Log,
		Controller: c.Controller,
	}
	hosts, err := parseHosts(c)
	if err != nil {
		return err
	}
	App.Hosts = hosts

	dnsCfg, err := parseDNS(c, hosts)
	if err != nil {
		return err
	}
	App.DNS = dnsCfg
	App.Users = parseAuthentication(c.Auth)
	App.Whitelist = parseWhitelist(c.WhiteList)
	return nil
}

func parseHosts(cfg *RawConfig) (*trie.DomainTrie, error) {
	tree := trie.New()

	// add default hosts
	if err := tree.Insert("localhost", net.IP{127, 0, 0, 1}); err != nil {
		logrus.Errorln("insert localhost to host error: ", err.Error())
	}

	if len(cfg.Hosts) != 0 {
		for domain, ipStr := range cfg.Hosts {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("%s is not a valid IP", ipStr)
			}
			_ = tree.Insert(domain, ip)
		}
	}

	return tree, nil
}

func hostWithDefaultPort(host string, defPort string) (string, error) {
	if !strings.Contains(host, ":") {
		host += ":"
	}

	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", err
	}

	if port == "" {
		port = defPort
	}

	return net.JoinHostPort(hostname, port), nil
}

func parseNameServer(servers []string) ([]dns.NameServer, error) {
	var nameservers []dns.NameServer
	for idx, server := range servers {
		// parse without scheme .e.g 8.8.8.8:53
		if !strings.Contains(server, "://") {
			server = "udp://" + server
		}
		u, err := url.Parse(server)
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		// parse with specific interface
		// .e.g 10.0.0.1#en0
		interfaceName := u.Fragment

		var addr, dnsNetType string
		switch u.Scheme {
		case "udp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "" // UDP
		case "tcp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "tcp" // TCP
		case "tls":
			addr, err = hostWithDefaultPort(u.Host, "853")
			dnsNetType = "tcp-tls" // DNS over TLS
		case "https":
			clearURL := url.URL{Scheme: "https", Host: u.Host, Path: u.Path}
			addr = clearURL.String()
			dnsNetType = "https" // DNS over HTTPS
		case "dhcp":
			addr = u.Host
			dnsNetType = "dhcp" // UDP from DHCP
		default:
			return nil, fmt.Errorf("DNS NameServer[%d] unsupport scheme: %s", idx, u.Scheme)
		}

		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		nameservers = append(
			nameservers,
			dns.NameServer{
				Net:       dnsNetType,
				Addr:      addr,
				Interface: interfaceName,
			},
		)
	}
	return nameservers, nil
}

func parseDNS(rawCfg *RawConfig, hosts *trie.DomainTrie) (*DNS, error) {
	cfg := rawCfg.DNS
	dnsCfg := &DNS{
		Enable: cfg.Enable,
		Listen: cfg.Listen,
		Port:   cfg.Port,
		Hosts:  hosts,
	}
	var err error
	if dnsCfg.NameServers, err = parseNameServer(cfg.NameServers); err != nil {
		return nil, err
	}

	return dnsCfg, nil
}

func parseAuthentication(rawRecords map[string]string) []auth.AuthUser {
	var users []auth.AuthUser
	for user, pass := range rawRecords {
		users = append(users, auth.AuthUser{User: user, Pass: pass})
	}
	return users
}

func parseWhitelist(IPCIDR []string) []net.IP {
	var ips []net.IP
	if len(IPCIDR) == 0 {
		return nil
	}
	for _, ipMask := range IPCIDR {
		if ipMask == "0.0.0.0" || ipMask == "::" || ipMask == "*" || ipMask == "all" {
			return nil
		}
		if ip := net.ParseIP(ipMask); ip != nil {
			ips = append(ips, ip)
			continue
		}
		if ip, _, _ := net.ParseCIDR(ipMask); ip != nil {
			ips = append(ips, ip)
		}
	}
	// access local ip address
	ifaces, err := iface.Interfaces()
	if err != nil {
		logrus.Fatalln(err)
	}
	for _, iface := range ifaces {
		for _, ipNet := range iface.Addrs {
			ips = append(ips, ipNet.IP)
		}
	}
	return ips
}
