package main

import (
	"context"
	"fmt"
	"github.com/common-nighthawk/go-figure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github/xmapst/mixed-socks/internal/conf"
	"github/xmapst/mixed-socks/internal/mixed"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"
)

var (
	ml     *mixed.Listener
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
		ml, err = mixed.New(context.Background(), conf.App.Host, conf.App.Port, conf.IPAuth)
		if err != nil {
			return
		}
		ml.ListenAndServe(conf.UserAuth)
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

func main() {
	cobra.CheckErr(cmd.Execute())
}

func registerSignalHandlers() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigs
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := ml.Shutdown(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}
	}()
}
