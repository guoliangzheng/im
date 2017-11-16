package service

import (
	"io/ioutil"
	"log"

	"fmt"

	"github.com/naoina/toml"
	"github.com/uber-go/zap"
)

type Config struct {
	Common struct {
		Version  string
		IsDebug  bool
		LogLevel string
		LogPath  string
	}

	Mysql struct {
		Addr     string
		Port     string
		Database string
		Acc      string
		Pw       string
	}
}

var Conf = &Config{}

func loadConfig(staticConf bool) {
	var contents []byte
	var err error

	if staticConf {
		//静态配置
		contents, err = ioutil.ReadFile("configs/center.toml")
	} else {
		contents, err = ioutil.ReadFile("/etc/gomqtt/center.toml")
	}

	if err != nil {
		log.Fatal("load config error", zap.Error(err))
	}

	tbl, err := toml.Parse(contents)
	if err != nil {
		log.Fatal("parse config error", zap.Error(err))
	}

	toml.UnmarshalTable(tbl, Conf)

	fmt.Println(Conf)

	// 初始化Logger
	InitLogger(Conf.Common.LogPath, Conf.Common.LogLevel, Conf.Common.IsDebug)
}
