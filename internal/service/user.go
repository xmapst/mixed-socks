package service

import (
	"github.com/sirupsen/logrus"
)

const userKeyPrefix = globalPrefix + ":user"

type User struct {
	Name   string   `json:"name" form:"name" binding:"required" description:"用户名" example:"name"`
	Pass   string   `json:"pass" form:"pass" description:"密码(sock4下不生效)" example:"123456"`
	CIDR   []string `json:"cidr" form:"cidr" description:"白名单" example:"0.0.0.0/0"`
	Remark string   `json:"remark" form:"remark" description:"备注" example:"小明"`
}

func SaveUser(auth *User) (err error) {
	var key = userKeyPrefix + ":" + auth.Name
	err = Set(key, auth)
	if err != nil {
		logrus.Error(err)
	}
	return
}

func DelUser(username string) (err error) {
	var key = userKeyPrefix + ":" + username
	err = Del(key)
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
	for _, v := range data {
		var _res User
		err = json.Unmarshal(v, &_res)
		if err != nil {
			logrus.Errorln(err)
			continue
		}
		res = append(res, _res)
	}
	return
}

func GetUser(username string) (res *User) {
	var key = userKeyPrefix + ":" + username
	data, err := Get(key)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	if data == nil {
		return
	}
	res = new(User)
	err = json.Unmarshal(data, res)
	if err != nil {
		logrus.Errorln(err)
	}
	return
}
