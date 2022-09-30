package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github/xmapst/mixed-socks/docs"
	"github/xmapst/mixed-socks/internal/conf"
	"github/xmapst/mixed-socks/internal/mixed"
	"math"
	"net/http"
	"time"
)

func Handler() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(cors.Default(), gin.Recovery(), gzip.Gzip(gzip.DefaultCompression), logger())
	engine.Use(func(c *gin.Context) {
		c.Header("Server", "Mixed-Socks")
		c.Header("X-Server", "Gin")
		c.Header("X-Powered-By", "XMapst")
		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Header("Pragma", "no-cache")
		c.Next()
	})
	engine.GET("/healthz", func(c *gin.Context) {
		c.SecureJSON(http.StatusOK, "running")
	})
	engine.GET("/version", version)
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	pprof.Register(engine)
	api := engine.Group("api")
	if conf.App.Auth != nil {
		api.Use(gin.BasicAuth(conf.App.Auth))
	}
	// api router path
	{
		api.GET("traffic", traffic)
		connection := api.Group("connections")
		{
			connection.GET("", connections)
			connection.POST("", closeAllConnections)
			connection.POST(":id", closeConnection)
		}
		server := api.Group("server")
		{
			// get server state
			server.GET("", state)
			// start/stop/restart server
			server.POST("", operate)
		}
		config := api.Group("config")
		{
			// detail
			config.GET("", getConf)
			// update
			config.POST("", saveConf)
		}
		auth := api.Group("auth")
		{
			// get auth state
			auth.GET("", getAuth)
			// enable auth
			auth.POST("", postAuth)
		}
		user := api.Group("user")
		{
			// list
			user.GET("", listUser)
			// add
			user.POST("", saveUser)
			// del
			user.POST(":username", delUser)
		}
	}
	// start socks server
	ml = mixed.New()
	err := ml.ListenAndServe()
	if err != nil {
		logrus.Fatalln(err)
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
