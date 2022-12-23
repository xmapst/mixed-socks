package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/constant"
	"os"
)

// Init prepare necessary files
func Init(dir string) error {
	// initial homedir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return fmt.Errorf("can't create config directory %s: %s", dir, err.Error())
		}
	}

	// initial config.yaml
	if _, err := os.Stat(constant.Path.Config()); os.IsNotExist(err) {
		logrus.Infoln("Can't find config, create a initial config file")
		f, err := os.OpenFile(constant.Path.Config(), os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("can't create file %s: %s", constant.Path.Config(), err.Error())
		}
		_, err = f.Write([]byte(defaultConf))
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

const defaultConf = `#support socks4, socks4a, socks5, socks5h, http proxy all in one
Inbound:
  Listen: 0.0.0.0
  Port: 8090

Controller:
  Enable: true
  Listen: 0.0.0.0
  Port: 8080

Log:
  Level: info

Hosts:
  '*.dev': 127.0.0.1
  'alpha.dev': '::1'

DNS:
  Enable: true
  Listen: 0.0.0.0
  Port: 53
  Nameservers:
    - 114.114.114.114
    - 8.8.8.8
    - tls://dns.rubyfish.cn:853
    - https://1.1.1.1/dns-query
`
