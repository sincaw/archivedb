package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel = zapcore.Level

const (
	LevelDebug = zapcore.DebugLevel
	LevelInfo  = zapcore.InfoLevel
	LevelWarn  = zapcore.WarnLevel
	LevelError = zapcore.ErrorLevel
	LevelFatal = zapcore.FatalLevel
)

var (
	globalLogger *logger
)

func init() {
	globalLogger = newLogger(zapcore.DebugLevel)
}

func newLogger(lv LogLevel) *logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return &logger{SugaredLogger: l.WithOptions(zap.IncreaseLevel(lv)).Sugar()}
}

type logger struct {
	*zap.SugaredLogger
}

func (l *logger) Warningf(s string, i ...interface{}) {
	l.SugaredLogger.Warnf(s, i)
}

func Logger() *logger {
	return globalLogger
}

func NewLogger(lv LogLevel) *logger {
	return newLogger(lv)
}
