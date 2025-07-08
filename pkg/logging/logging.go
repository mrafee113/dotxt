package logging

import (
	"dotxt/config"
	"dotxt/pkg/utils"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger       *zap.SugaredLogger
	fileHandle   *os.File
	fileCore     *zapcore.Core
	ConsoleLevel int
	consoleCore  *zapcore.Core
)

func Initialize() error {
	consoleEnc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	consoleCore = utils.MkPtr(zapcore.NewCore(
		consoleEnc,
		zapcore.Lock(os.Stdout),
		zapcore.Level(ConsoleLevel),
	))

	fileEncCfg := zap.NewProductionEncoderConfig()
	fileEncCfg.TimeKey = "ts"
	fileEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEnc := zapcore.NewConsoleEncoder(fileEncCfg)

	filepath := filepath.Join(config.ConfigPath(), "log")
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fileHandle = f
	fileSync := zapcore.AddSync(io.Writer(f))

	fileLevel := viper.GetInt("logging.file-level")
	fileCore = utils.MkPtr(zapcore.NewCore(
		fileEnc,
		fileSync,
		zapcore.Level(fileLevel),
	))

	core := zapcore.NewTee(*consoleCore, *fileCore)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Logger = logger.Sugar()
	return nil
}

func Close() error {
	if fileHandle != nil {
		return fileHandle.Close()
	}
	return nil
}
