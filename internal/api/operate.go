package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/mixed"
)

var ml = &mixed.Listener{}

// State
// @Summary State
// @description mixed socks server state
// @Tags Server
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/server [get]
func state(c *gin.Context) {
	render := Gin{c}
	render.SetJson(ml.State())
}

// Operate
// @Summary Operate
// @description Operate mixed socks server
// @Tags Server
// @Security BasicAuth
// @Param operate query string false "operate [start,stop,restart]"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/server [post]
func operate(c *gin.Context) {
	render := Gin{c}
	o := c.DefaultQuery("operate", "start")
	switch o {
	case "stop":
		if !ml.State() {
			render.SetJson(ml.State())
			return
		}
		err := ml.Shutdown()
		if err != nil {
			render.SetError(CodeErrNotRunning, err)
			return
		}
	case "restart":
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
	default:
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

	}
	render.SetJson(ml.State())
}
