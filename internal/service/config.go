package service

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Conf struct {
	Host    string   `json:"" form:"Host" description:"监听地址" example:"0.0.0.0"`
	Port    int64    `json:"" form:"Port" description:"监听端口" example:"8090"`
	CIDR    []string `json:"" form:"CIDR" description:"白名单" example:"0.0.0.0/0"`
	Timeout string   `json:"" form:"Timeout" description:"超时时间" example:"30s"`
}

func (c *Conf) Save() error {
	err := set(ConfigTablePrefix, c)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	return nil
}

func (c *Conf) Get() *Conf {
	value, err := get(ConfigTablePrefix)
	if err != nil {
		logrus.Errorln(err)
		return c
	}
	if value == nil {
		return c
	}
	err = json.Unmarshal(value, c)
	if err != nil {
		logrus.Errorln(err)
	}
	return c
}

func (c *Conf) ParseTimeout() time.Duration {
	t, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return t
}
