package api

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/statistic"
	"net/http"
	"time"
)

type Traffic struct {
	Up   int64 `json:""`
	Down int64 `json:""`
}

// get
// @Summary get
// @description get traffic
// @Tags Traffic
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/traffic [get]
func traffic(c *gin.Context) {
	var render = Gin{c}
	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(c.Request) {
		var err error
		wsConn, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
	}
	t := statistic.Manager
	if wsConn == nil {
		c.Writer.Header().Set("Content-Type", "application/json")
		up, down := t.Now()
		render.SetJson(Traffic{
			Up:   up,
			Down: down,
		})
		return
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	buf := &bytes.Buffer{}
	var err error
	for range tick.C {
		buf.Reset()
		up, down := t.Now()
		if err := json.NewEncoder(buf).Encode(Traffic{
			Up:   up,
			Down: down,
		}); err != nil {
			break
		}

		if wsConn == nil {
			_, err = c.Writer.Write(buf.Bytes())
			c.Writer.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}
		if err != nil {
			logrus.Error(err)
			break
		}
	}
}
