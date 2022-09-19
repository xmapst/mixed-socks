package service

import "github.com/sirupsen/logrus"

type Conf struct {
	Host string   `json:"host" description:"监听地址" example:"0.0.0.0"`
	Port int      `json:"port" description:"监听端口" example:"8090"`
	CIDR []string `json:"cidr" description:"白名单" example:"0.0.0.0/0"`
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
		Host: "0.0.0.0",
		Port: 8090,
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
