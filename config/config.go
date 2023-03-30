package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

var C Config

type Config struct {
	Log Log `mapstructure:"log"`
}

type Log struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

func ParseConfig(configPath string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if configPath == "" {
		viper.AddConfigPath(".")
	} else {
		viper.SetConfigFile(configPath)
	}

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("fatal error config file: %s", err)
	}

	if err := viper.Unmarshal(&C); err != nil {
		return err
	}

	if C.Log.Level == "debug" {
		log.Printf("Following configuration is loaded:\n%+v\n", C)
	}

	return nil
}
