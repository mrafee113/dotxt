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
	Logger *zap.SugaredLogger

	FileLevel  int
	fileHandle *os.File
	fileCore   *zapcore.Core
	fileEnc    zapcore.Encoder

	ConsoleLevel int
	consoleCore  *zapcore.Core
	consoleEnc   = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
)

func init() {
	initFileEnc()
	InitConsole()
	Initialize()
}

func InitConsole() {
	consoleCore = utils.MkPtr(zapcore.NewCore(
		consoleEnc,
		zapcore.Lock(os.Stdout),
		zapcore.Level(ConsoleLevel),
	))
}

func initFileEnc() {
	fileEncCfg := zap.NewProductionEncoderConfig()
	fileEncCfg.TimeKey = "ts"
	fileEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEnc = zapcore.NewConsoleEncoder(fileEncCfg)
}

func InitFile() error {
	filepath := filepath.Join(config.ConfigPath(), "log")
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fileHandle = f
	fileSync := zapcore.AddSync(io.Writer(f))

	FileLevel = viper.GetInt("logging.file-level")
	fileCore = utils.MkPtr(zapcore.NewCore(
		fileEnc,
		fileSync,
		zapcore.Level(FileLevel),
	))
	return nil
}

func Initialize() {
	var core zapcore.Core
	if fileCore != nil {
		core = zapcore.NewTee(*consoleCore, *fileCore)
	} else {
		core = zapcore.NewTee(*consoleCore)
	}
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.PanicLevel))
	Logger = logger.Sugar()
}

func Close() error {
	if fileHandle != nil {
		return fileHandle.Close()
	}
	return nil
}
