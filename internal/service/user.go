package service

import (
	"github.com/sirupsen/logrus"
)

type Auth struct {
	Enabled bool `json:"enabled" form:"name" description:"启用" example:"false"`
}

func (a *Auth) Save() (err error) {
	var key = UserTablePrefix + "_" + "auth"
	err = set(key, a)
	if err != nil {
		logrus.Error(err)
	}
	return
}

func (a *Auth) Get() bool {
	var key = UserTablePrefix + "_" + "auth"
	data, err := get(key)
	if err != nil {
		logrus.Errorln(err)
		return false
	}
	if data == nil {
		return false
	}
	err = json.Unmarshal(data, a)
	if err != nil {
		logrus.Errorln(err)
	}
	return a.Enabled
}

type User struct {
	Name     string   `json:"name" form:"name" binding:"required" description:"用户名" example:"name"`
	Pass     string   `json:"pass" form:"pass" description:"密码(sock4下不生效)" example:"123456"`
	CIDR     []string `json:"cidr" form:"cidr" description:"白名单" example:"0.0.0.0/0"`
	Remark   string   `json:"remark" form:"remark" description:"备注" example:"小明"`
	Disabled bool     `json:"disabled" form:"disabled" description:"禁用" example:"false"`
}

func (u *User) Save() (err error) {
	var key = UserTablePrefix + ":" + u.Name
	err = set(key, u)
	if err != nil {
		logrus.Error(err)
	}
	return
}

func (u *User) Delete() (err error) {
	var key = UserTablePrefix + ":" + u.Name
	err = del(key)
	if err != nil {
		logrus.Errorln(err)
	}
	return
}

func (u *User) List() (res []User, err error) {
	var data = list(UserTablePrefix)
	if data == nil {
		return nil, nil
	}
	for _, v := range data {
		var _res User
		err = json.Unmarshal(v, &_res)
		if err != nil || _res.Name == "" {
			continue
		}
		res = append(res, _res)
	}
	return
}

func (u *User) Get() (*User, error) {
	var key = UserTablePrefix + ":" + u.Name
	data, err := get(key)
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	err = json.Unmarshal(data, u)
	if err != nil {
		logrus.Errorln(err)
	}
	return u, err
}
