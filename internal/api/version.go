package api

import (
	"github.com/gin-gonic/gin"
	info "github/xmapst/mixed-socks"
	"net/http"
	"runtime"
)

type Info struct {
	Version string `json:",omitempty"`
	Go      struct {
		Version string `json:",omitempty"`
		OS      string `json:",omitempty"`
		Arch    string `json:",omitempty"`
	} `json:",omitempty"`
	Git struct {
		Url    string `json:",omitempty"`
		Branch string `json:",omitempty"`
		Commit string `json:",omitempty"`
	} `json:",omitempty"`
	User struct {
		Name  string `json:",omitempty"`
		Email string `json:",omitempty"`
	} `json:",omitempty"`
	BuildTime string `json:",omitempty"`
}

// Version
// @Summary Version
// @description 当前服务器版本
// @Tags Version
// @Success 200 {object} Info
// @Failure 500 {object} JSONResult
// @Router /version [get]
func version(c *gin.Context) {
	c.JSON(http.StatusOK, Info{
		Version: info.Version,
		Go: struct {
			Version string `json:",omitempty"`
			OS      string `json:",omitempty"`
			Arch    string `json:",omitempty"`
		}(struct {
			Version string
			OS      string
			Arch    string
		}{
			Version: runtime.Version(),
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
		}),
		Git: struct {
			Url    string `json:",omitempty"`
			Branch string `json:",omitempty"`
			Commit string `json:",omitempty"`
		}(struct {
			Url    string
			Branch string
			Commit string
		}{
			Url:    info.GitUrl,
			Branch: info.GitBranch,
			Commit: info.GitCommit,
		}),
		User: struct {
			Name  string `json:",omitempty"`
			Email string `json:",omitempty"`
		}(struct {
			Name  string
			Email string
		}{
			Name:  info.UserName,
			Email: info.UserEmail,
		}),
		BuildTime: info.BuildTime,
	})
}
