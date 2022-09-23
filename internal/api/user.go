package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github/xmapst/mixed-socks/internal/service"
)

// list
// @Summary list
// @description List all user
// @Tags User
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/user [get]
func listUser(c *gin.Context) {
	render := Gin{Context: c}
	user := &service.User{}
	res, err := user.List()
	if err != nil {
		render.SetError(CodeErrMsg, err)
		return
	}
	render.SetJson(res)
}

// save
// @Summary save
// @description create or update user
// @Tags User
// @Security BasicAuth
// @Param user body service.User true "user"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/user [post]
func saveUser(c *gin.Context) {
	render := Gin{Context: c}
	var req = &service.User{}
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

// delete
// @Summary delete
// @description delete user
// @Tags User
// @Security BasicAuth
// @Param username path string true "username"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/user/{username} [post]
func delUser(c *gin.Context) {
	render := Gin{Context: c}
	username := c.Param("username")
	if username == "" || username == ":username" {
		render.SetError(CodeErrParam, errors.New("missing id parameter"))
		return
	}
	user := &service.User{
		Name: username,
	}
	err := user.Delete()
	if err != nil {
		render.SetError(CodeErrMsg, err)
		return
	}
	render.SetJson("deleted")
}
