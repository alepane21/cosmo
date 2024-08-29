package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func NewObserver() (*zap.Logger, *observer.ObservedLogs) {
	core, observedLogs := observer.New(zap.DebugLevel)
	return zap.New(core), observedLogs
}
