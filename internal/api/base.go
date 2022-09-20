package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const (
	CodeSuccess = 0
	CodeErrApp  = iota + 1000
	CodeErrMsg
	CodeErrParam
	CodeErrNoPriv
	CodeErrNotRunning
)

var MsgFlags = map[int]string{
	CodeErrApp:        "内部错误",
	CodeSuccess:       "成功",
	CodeErrMsg:        "未知错误",
	CodeErrParam:      "参数错误",
	CodeErrNoPriv:     "沒有权限",
	CodeErrNotRunning: "没有启动",
}

// getMsg get error information based on Code
func getMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}

	return MsgFlags[CodeErrApp]
}

type Gin struct {
	*gin.Context
}

type JSONResult struct {
	Code    int         `json:"code" description:"返回码" example:"0000"`
	Message string      `json:"message,omitempty" description:"消息" example:"消息"`
	Data    interface{} `json:"data,omitempty" description:"数据"`
}

func newRes(data interface{}, err error, code int) *JSONResult {
	if code == 200 {
		code = 0
	}
	codeMsg := ""
	if code != 0 {
		codeMsg = getMsg(code)
	}

	return &JSONResult{
		Data: data,
		Code: code,
		Message: func() string {
			result := func() string {
				if err == nil {
					return ""
				}
				return err.Error()
			}()
			if codeMsg != "" && result != "" {
				result += ", " + codeMsg
			} else if codeMsg != "" {
				result = codeMsg
			}
			return strings.TrimSpace(result)
		}(),
	}
}

// SetRes Response res
func (g *Gin) SetRes(res interface{}, err error, code int) {
	g.SecureJSON(http.StatusOK, newRes(res, err, code))
}

// SetJson Set Json
func (g *Gin) SetJson(res interface{}) {
	g.SetRes(res, nil, CodeSuccess)
}

// SetError Check Error
func (g *Gin) SetError(code int, err error) {
	g.SetRes(nil, err, code)
	g.Abort()
}
