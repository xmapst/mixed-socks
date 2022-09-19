package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/mixed"
	"github/xmapst/mixed-socks/internal/service"
)

var ml = &mixed.Listener{}

// state
// @Summary state
// @description get mixed socks state
// @Tags Operate
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/state [get]
func state(c *gin.Context) {
	render := Gin{c}
	if !ml.Running() {
		render.SetError(CodeErrNotRunning, errors.New("not running"))
		return
	}
	render.SetJson("is running")
}

// start
// @Summary start
// @description start mixed socks
// @Tags Operate
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/start [post]
func start(c *gin.Context) {
	render := Gin{c}
	if ml.Running() {
		render.SetError(CodeSuccess, errors.New("is running"))
		return
	}
	res := service.GetConf()
	ml = mixed.New(res.Host, res.Port)
	go func() {
		err := ml.ListenAndServe()
		if err != nil {
			logrus.Errorln(err)
		}
	}()
	render.SetJson("started")
}

// stop
// @Summary stop
// @description stop mixed socks
// @Tags Operate
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/stop [post]
func stop(c *gin.Context) {
	render := Gin{c}
	if !ml.Running() {
		render.SetError(CodeErrNotRunning, errors.New("not running"))
		return
	}
	err := ml.Shutdown()
	if err != nil {
		render.SetError(CodeErrNotRunning, err)
		return
	}
	render.SetJson("stopped")
}

// reload
// @Summary reload
// @description reload mixed socks config
// @Tags Operate
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/reload [post]
func reload(c *gin.Context) {
	render := Gin{c}
	if ml.Running() {
		err := ml.Shutdown()
		if err != nil {
			render.SetError(CodeErrMsg, err)
			return
		}
	}

	res := service.GetConf()
	ml = mixed.New(res.Host, res.Port)
	go func() {
		err := ml.ListenAndServe()
		if err != nil {
			logrus.Errorln(err)
		}
	}()
	render.SetJson("reload")
}
