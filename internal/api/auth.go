package api

import (
	"github.com/gin-gonic/gin"
	"github/xmapst/mixed-socks/internal/service"
)

// Auth
// @Summary Auth
// @description enable user auth
// @Tags Auth
// @Security BasicAuth
// @Param auth body service.Auth true "user auth"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/auth [post]
func postAuth(c *gin.Context) {
	render := Gin{Context: c}
	var req = &service.Auth{}
	err := c.ShouldBind(req)
	if err != nil {
		render.SetError(CodeErrParam, err)
		return
	}
	err = req.Save()
	if err != nil {
		render.SetError(CodeErrMsg, err)
		return
	}
	render.SetJson("saved")
}

// Auth
// @Summary Auth
// @description get user auth state
// @Tags Auth
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/auth [get]
func getAuth(c *gin.Context) {
	render := Gin{Context: c}
	auth := &service.Auth{}
	render.SetJson(auth.Get())
}
