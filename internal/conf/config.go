package conf

import (
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github/xmapst/mixed-socks/internal/auth"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var (
	App       *Config
	Path      string
	LogOutput *lumberjack.Logger
	IPAuth    = &auth.IPAuth{}
	UserAuth  = &auth.PasswordAuth{}
)

type Config struct {
	Host      string      `yaml:""`
	Port      int         `yaml:""`
	Log       Log         `yaml:""`
	Timeout   string      `yaml:""`
	WhiteList []string    `yaml:""`
	Users     []auth.User `yaml:""`
}

type Log struct {
	Filename   string `yaml:""`
	Level      string `yaml:",default=info"`
	MaxBackups int    `yaml:",default=7"`
	MaxSize    int    `yaml:",default=500"`
	MaxAge     int    `yaml:",default=28"`
	Compress   bool   `yaml:",default=true"`
}

func viperLoadConf() error {
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	var conf = &Config{
		Log: Log{
			Level:      "info",
			MaxBackups: 7,
			MaxSize:    500,
			MaxAge:     28,
			Compress:   true,
		},
	}
	err = viper.Unmarshal(conf)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	App = conf
	return nil
}

func Load() error {
	viper.SetConfigFile(Path)
	err := viperLoadConf()
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logrus.Infoln(e.Name, "config file modified")
		err = viperLoadConf()
		if err != nil {
			return
		}
		err = App.reload()
		if err != nil {
			return
		}
	})

	err = App.reload()
	if err != nil {
		return err
	}
	c := cron.New()
	_, _ = c.AddFunc("@daily", func() {
		if LogOutput != nil {
			_ = LogOutput.Rotate()
		}
	})
	c.Start()
	return nil
}

func (c *Config) reload() error {
	level, err := logrus.ParseLevel(c.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	if LogOutput != nil {
		err = LogOutput.Rotate()
		if err != nil {
			logrus.Warningln(err)
		}
	}
	if c.Log.Filename != "" {
		LogOutput = &lumberjack.Logger{
			Filename:   c.Log.Filename,
			MaxBackups: c.Log.MaxBackups,
			MaxSize:    c.Log.MaxSize,  // megabytes
			MaxAge:     c.Log.MaxAge,   // days
			Compress:   c.Log.Compress, // disabled by default
			LocalTime:  true,           // use local time zone
		}
		logrus.SetOutput(LogOutput)
	} else {
		LogOutput = nil
		logrus.SetOutput(os.Stdout)
	}

	if c.WhiteList != nil {
		err = IPAuth.New(c.WhiteList)
		if err != nil {
			logrus.Errorln(err)
			return err
		}
	}

	if c.Users != nil {
		err = UserAuth.New(c.Users)
		if err != nil {
			logrus.Errorln(err)
			return err
		}
	}
	return nil
}

func (c *Config) ParseTimeout() time.Duration {
	t, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return t
}
