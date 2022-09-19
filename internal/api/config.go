package api

import (
	"github.com/gin-gonic/gin"
	"github/xmapst/mixed-socks/internal/service"
	"net/http"
)

// get
// @Summary get
// @description get mixed socks config
// @Tags Config
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/config [get]
func getConf(c *gin.Context) {
	res := service.GetConf()
	c.SecureJSON(http.StatusOK, res)
}

// save
// @Summary save
// @description update mixed socks config
// @Tags Config
// @Security BasicAuth
// @Param scripts body service.Conf true "config"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/config [post]
func saveConf(c *gin.Context) {
	var render = Gin{c}
	var req = &service.Conf{}
	err := c.ShouldBind(req)
	if err != nil {
		render.SetError(CodeErrParam, err)
		return
	}
	err = service.SaveConf(req)
	if err != nil {
		render.SetError(CodeErrParam, err)
		return
	}
	render.SetJson("saved")
}
