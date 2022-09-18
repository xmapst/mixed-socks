package service

import (
	"github.com/sirupsen/logrus"
)

const userKeyPrefix = globalPrefix + ":user"

type User struct {
	Username string   `xml:"" yaml:"" json:"" form:"" binding:"required"`
	Password string   `xml:"" yaml:"" json:"" form:""`
	CIDR     []string `xml:"" yaml:"" json:"" form:""`
	Remark   string   `xml:"" yaml:"" json:"" form:""`
}

func SaveUser(user *User) (err error) {
	var key = userKeyPrefix + ":" + user.Username
	err = Set(key, user)
	if err != nil {
		logrus.Error(err)
	}
	return
}

func DelUser(id string) (err error) {
	err = Del(id)
	if err != nil {
		logrus.Errorln(err)
	}
	return
}

func ListUser() (res []User, err error) {
	var data = List(userKeyPrefix)
	if data == nil {
		return nil, nil
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		logrus.Errorln(err)
	}
	return
}

func GetUser(username string) *User {
	var key = userKeyPrefix + ":" + username
	data, err := Get(key)
	if err != nil {
		logrus.Errorln(err)
		return nil
	}
	var res = &User{}
	err = json.Unmarshal(data, res)
	if err != nil {
		logrus.Errorln(err)
	}
	return res
}
