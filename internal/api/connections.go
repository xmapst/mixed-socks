package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github/xmapst/mixed-socks/internal/statistic"
	"net/http"
	"strconv"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// get
// @Summary get
// @description get connections
// @Tags Connections
// @Security BasicAuth
// @Param interval query string false "interval time"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/connections [get]
func connections(c *gin.Context) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		snapshot := statistic.Manager.Snapshot()
		c.SecureJSON(http.StatusOK, snapshot)
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	intervalStr := c.DefaultQuery("interval", "600")
	interval := 1000
	if intervalStr != "" {
		t, err := strconv.Atoi(intervalStr)
		if err != nil {
			c.SecureJSON(http.StatusBadRequest, err)
			return
		}

		interval = t
	}

	buf := &bytes.Buffer{}
	sendSnapshot := func() error {
		buf.Reset()
		snapshot := statistic.Manager.Snapshot()
		if err := json.NewEncoder(buf).Encode(snapshot); err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, buf.Bytes())
	}
	if err := sendSnapshot(); err != nil {
		return
	}
	tick := time.NewTicker(time.Millisecond * time.Duration(interval))
	defer tick.Stop()
	for range tick.C {
		if err := sendSnapshot(); err != nil {
			break
		}
	}
}

// close all connections
// @Summary close all connections
// @description close all connections
// @Tags Connections
// @Security BasicAuth
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/connections [post]
func closeAllConnections(c *gin.Context) {
	var render = Gin{c}
	snapshot := statistic.Manager.Snapshot()
	for _, c := range snapshot.Connections {
		_ = c.Close()
	}
	render.SetJson("closed")
}

// close
// @Summary close
// @description close connections
// @Tags Connections
// @Security BasicAuth
// @Param id path string true "connections id"
// @Success 200 {object} JSONResult{}
// @Failure 500 {object} JSONResult{}
// @Router /api/connections/{id} [post]
func closeConnection(c *gin.Context) {
	var render = Gin{c}
	id := c.Param("id")
	if id == "" || id == ":id" {
		render.SetError(CodeErrParam, errors.New("missing id parameter"))
		return
	}
	snapshot := statistic.Manager.Snapshot()
	for _, c := range snapshot.Connections {
		if c.ID() == id {
			_ = c.Close()
			break
		}
	}
	render.SetJson("closed")
}
