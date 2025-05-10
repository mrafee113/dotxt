package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"to-dotxt/pkg/terrors"
	"to-dotxt/pkg/utils"

	"github.com/spf13/viper"
)

const (
	EnvPrefix = "DOTXT"
	EnvCFG    = "DOTXT_CONFIG"
)

var DefaultPath = "~/.to-dotxt"

var configPath string

func ConfigPath() string {
	return configPath
}
func setConfigPath(path string) error {
	path, err := utils.NormalizePath(path)
	if err != nil {
		return err
	}
	configPath = path
	return nil
}

func SelectConfigFile(arg string) error {
	var path string
	env := os.Getenv(EnvCFG)
	if arg != "" {
		path = arg
	} else if env != "" {
		path = env
	} else {
		path = DefaultPath
	}

	setConfigPath(path)
	return nil
}

func InitViper(arg string) error {
	err := SelectConfigFile(arg)
	if err != nil {
		return err
	}
	path := ConfigPath()
	viper.SetConfigType("yaml")
	viper.SetConfigName("dotxt")
	viper.AddConfigPath(path)
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	err = viper.ReadConfig(bytes.NewReader([]byte(DefaultConfig)))
	if err != nil {
		return fmt.Errorf("%w: failed parsing default configurations: %w", terrors.ErrParse, err)
	}
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	err = viper.SafeWriteConfigAs(filepath.Join(path, "dotxt.yaml"))
	if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
		return nil
	}
	return err
}
