package main

import (
	"context"
	"fmt"
	"github.com/common-nighthawk/go-figure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github/xmapst/mixed-socks/internal/api"
	"github/xmapst/mixed-socks/internal/conf"
	"github/xmapst/mixed-socks/internal/service"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"
)

var (
	//ml     *mixed.Listener
	server *http.Server
	Header = figure.NewFigure("MixedSocks", "doom", true).String()
)
var cmd = &cobra.Command{
	Use:               os.Args[0],
	Short:             "Support socks4, socks4a, socks5, socks5h, http proxy all in one",
	DisableAutoGenTag: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(Header)
		err := conf.Load()
		if err != nil {
			return
		}
		err = service.New(conf.App.DataDir)
		if err != nil {
			return
		}
		time.Sleep(1 * time.Second)
		// start http server
		server = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", conf.App.Host, conf.App.Port),
			WriteTimeout: time.Second * 180,
			ReadTimeout:  time.Second * 180,
			IdleTimeout:  time.Second * 180,
			Handler:      api.Handler(),
		}
		logrus.Infof("start http server, listen %s", server.Addr)
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logrus.Error(err)
		}
		select {}
	},
}

func init() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:03:04",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			file = fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
			function = path.Base(frame.Function)
			return
		},
	})
	registerSignalHandlers()
	cmd.PersistentFlags().StringVarP(&conf.Path, "config", "c", "config.yaml", "config file path")
}

// @title Mixed-Socks
// @description support socks4, socks5, http proxy all in one
// @version v2.0.0
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
func main() {
	cobra.CheckErr(cmd.Execute())
}

func registerSignalHandlers() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigs
		//err := ml.Shutdown()
		logrus.Infoln("received signal, exiting...")
		if server != nil {
			// 关闭http server
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			err := server.Shutdown(shutdownCtx)
			if err != nil {
				logrus.Error(err)
			}
		}
		service.Close()
		os.Exit(0)
	}()
}
