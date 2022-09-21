package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/mixed"
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
	render.SetJson(ml.State())
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
	if ml.State() {
		render.SetJson(ml.State())
		return
	}
	ml = mixed.New()
	go func() {
		err := ml.ListenAndServe()
		if err != nil {
			logrus.Errorln(err)
		}
	}()
	render.SetJson(ml.State())
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
	if !ml.State() {
		render.SetJson(ml.State())
		return
	}
	err := ml.Shutdown()
	if err != nil {
		render.SetError(CodeErrNotRunning, err)
		return
	}
	render.SetJson(ml.State())
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
	if ml.State() {
		err := ml.Shutdown()
		if err != nil {
			render.SetError(CodeErrMsg, err)
			return
		}
	}
	ml = mixed.New()
	err := ml.ListenAndServe()
	if err != nil {
		render.SetError(CodeErrMsg, err)
	}
	render.SetJson(ml.State())
}
