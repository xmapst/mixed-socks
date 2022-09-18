package service

import "github.com/sirupsen/logrus"

type Conf struct {
	Host string   `xml:"" yaml:"" json:""`
	Port int      `xml:"" yaml:"" json:""`
	CIDR []string `xml:"" yaml:"" json:""`
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

func GetConf() *Conf {
	conf := &Conf{
		Host: "0.0.0.0",
		Port: 8090,
	}
	value, err := Get(configKey)
	if err != nil {
		logrus.Errorln(err)
		return conf
	}
	err = json.Unmarshal(value, conf)
	if err != nil {
		logrus.Errorln(err)
	}
	return conf
}
