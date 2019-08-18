package dlog

import (
	"sync"

	"github.com/sirupsen/logrus" //nolint:depguard
)

var (
	defaultLogger     Logger
	defaultLoggerOnce sync.Once
)

func getDefaultLogger() Logger {
	defaultLoggerOnce.Do(func() { defaultLogger = WrapLogrus(logrus.New()) })
	return defaultLogger
}
