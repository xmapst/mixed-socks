package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github/xmapst/mixed-socks/docs"
	"math"
	"net/http"
	"time"
)

func Handler() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(cors.Default(), gin.Recovery(), logger())
	engine.GET("healthz", func(c *gin.Context) {
		c.SecureJSON(http.StatusOK, "running")
	})
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	api := engine.Group("api")
	{
		api.GET("state", state)
		api.POST("start", start)
		api.POST("stop", stop)
		api.POST("reload", reload)
		config := api.Group("config")
		{
			// detail
			config.GET("", getConf)
			// update
			config.PUT("", saveConf)
		}
		user := api.Group("user")
		{
			// list
			user.GET("", listUser)
			// add
			user.POST("", saveUser)
			// del
			user.DELETE(":username", delUser)
		}
	}
	return engine
}

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		method := c.Request.Method
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := logrus.WithFields(logrus.Fields{
			"status_code": statusCode,
			"latency":     latency, // time to process
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"referer":     referer,
			"length":      dataLength,
			"user_agent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			entry.Info("none")
		}
	}
}
