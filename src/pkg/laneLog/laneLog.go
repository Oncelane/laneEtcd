package laneLog

import (
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// var Logger *zap.SugaredLogger

var Logger *zap.SugaredLogger

func init() {
	InitLogger("log", true)
}

func InitLogger(name string, enableConsole bool) {
	logDir := "logs"                                  // 日志目录,不存在则创建
	if err := os.MkdirAll(logDir, 0755); err != nil { // 创建日志目录
		panic(err)
	}
	logFile := filepath.Join(logDir, name) // 日志文件,不存在则创建
	encoder := getFileEncoder()
	writeSyncer := getLogWriter(logFile)
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	if enableConsole {
		consoleSyncer := zapcore.Lock(os.Stdout)
		core = zapcore.NewTee(
			core,
			zapcore.NewCore(getConsoleEncoder(), consoleSyncer, zapcore.DebugLevel),
		)
	}
	Logger = zap.New(core, zap.AddCaller()).Sugar()
}

func getConsoleEncoder() zapcore.Encoder {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05")
	encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderCfg)
}
func getFileEncoder() zapcore.Encoder {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewConsoleEncoder(encoderCfg)
}

func getLogWriter(filename string) zapcore.WriteSyncer {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	return zapcore.AddSync(lumberjackLogger)
}
