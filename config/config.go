package config

import (
	"flag"
	"log"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Version 版本号
const Version = "0.0.1"

var cfgPath = ""

// LatestConfig 最新读取配置
var LatestConfig Config

var readFlag bool = false

// ReadConf 读取配置
func ReadConf() Config {
	if cfgPath == "" {
		flag.StringVar(&cfgPath, "config", "config.toml", "配置文件位置")
		flag.Parse()
		initCfg()
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("Error reading config file: %v\n", err)
	}
	err = toml.Unmarshal(data, &LatestConfig)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v\n", err)
	}
	return LatestConfig
}
