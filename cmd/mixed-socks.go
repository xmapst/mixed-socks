package main

import (
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks"
	"github.com/xmapst/mixed-socks/internal/config"
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/engine"
	"go.uber.org/automaxprocs/maxprocs"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	flagset    map[string]bool
	configFile string
	version    bool
)

func init() {
	flag.StringVar(&configFile, "c", "", "specify configuration file")
	flag.BoolVar(&version, "v", false, "show current version")
	flag.Parse()

	flagset = map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		flagset[f.Name] = true
	})

	logrus.SetReportCaller(true)
	logrus.SetFormatter(&ConsoleFormatter{})
}

func main() {
	_, err := maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))
	if err != nil {
		logrus.Fatalln(err.Error())
	}

	if version {
		fmt.Println(info.VersionInfo())
		return
	}

	if configFile != "" {
		if !filepath.IsAbs(configFile) {
			currentDir, _ := os.Getwd()
			configFile = filepath.Join(currentDir, configFile)
		}
		constant.SetConfig(configFile)
	} else {
		configFile = filepath.Join(constant.Path.HomeDir(), constant.Path.Config())
		constant.SetConfig(configFile)
	}
	info.PrintHeadInfo()
	if err := config.Init(constant.Path.HomeDir()); err != nil {
		logrus.Fatalf("Initial configuration directory error: %s", err.Error())
	}

	if err := engine.Run(); err != nil {
		logrus.Fatalf("Parse config error: %s", err.Error())
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

type ConsoleFormatter struct {
	logrus.TextFormatter
}

func (c *ConsoleFormatter) TrimFunctionSuffix(s string) string {
	if strings.Contains(s, ".func") {
		index := strings.Index(s, ".func")
		s = s[:index]
	}
	return s
}

func (c *ConsoleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	file := path.Base(entry.Caller.File)
	function := c.TrimFunctionSuffix(path.Base(entry.Caller.Function))
	logStr := fmt.Sprintf("%s %s %s:%d %s %v\n",
		entry.Time.Format("2006/01/02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		file,
		entry.Caller.Line,
		function,
		entry.Message,
	)
	return []byte(logStr), nil
}
