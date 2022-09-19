package service

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Conf struct {
	Host    string   `json:"host" description:"监听地址" example:"0.0.0.0"`
	Port    int      `json:"port" description:"监听端口" example:"8090"`
	CIDR    []string `json:"cidr" description:"白名单" example:"0.0.0.0/0"`
	Timeout string   `json:"timeout" description:"超时时间" example:"30s"`
}

const configKey = globalPrefix + ":config"

func SaveConf(data *Conf) error {
	err := Set(configKey, data)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	return nil
}

func GetConf() (conf *Conf) {
	conf = &Conf{
		Host:    "0.0.0.0",
		Port:    8090,
		Timeout: "30s",
	}
	value, err := Get(configKey)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	if value == nil {
		return
	}
	conf = new(Conf)
	err = json.Unmarshal(value, conf)
	if err != nil {
		logrus.Errorln(err)
	}
	return
}

func (c *Conf) ParseTimeout() time.Duration {
	t, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return t
}
