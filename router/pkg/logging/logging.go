package logging

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	requestIDField = "reqId"
)

type RequestIDKey struct{}

func New(prettyLogging bool, debug bool, levelOutput zapcore.Level, fileOutput string, fileLevel zapcore.Level) *zap.Logger {
	cores := []zapcore.Core{newZapOutputLogger(prettyLogging, levelOutput)}
	if fileOutput != "" {
		cores = append(cores, newZapFileLogger(fileOutput, fileLevel))
	}

	return newZapLogger(
		debug,
		cores...,
	)
}

func zapBaseEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.SecondsDurationEncoder
	ec.TimeKey = "time"
	return ec
}

func ZapJsonEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		nanos := t.UnixNano()
		millis := int64(math.Trunc(float64(nanos) / float64(time.Millisecond)))
		enc.AppendInt64(millis)
	}
	return zapcore.NewJSONEncoder(ec)
}

func zapConsoleEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.ConsoleSeparator = " "
	ec.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05 PM")
	ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(ec)
}

func attachBaseFields(core zapcore.Core) zapcore.Core {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	core = core.With(
		[]zapcore.Field{{
			Key:    "hostname",
			Type:   zapcore.StringType,
			String: host,
		}, {
			Key:     "pid",
			Type:    zapcore.Int64Type,
			Integer: int64(os.Getpid()),
		}},
	)

	return core
}

func newZapFileLogger(file string, level zapcore.Level) zapcore.Core {
	fileOpen, closer, err := zap.Open(file)
	if err != nil && closer != nil {
		closer()
	}
	if err != nil {
		log.Fatalf("could not open log file: %s\n", err)
	}

	core := zapcore.NewCore(ZapJsonEncoder(), fileOpen, level)

	return attachBaseFields(core)
}

func newZapOutputLogger(prettyLogging bool, level zapcore.Level) zapcore.Core {
	var encoder zapcore.Encoder
	if prettyLogging {
		encoder = zapConsoleEncoder()
	} else {
		encoder = ZapJsonEncoder()
	}

	syncer := zapcore.AddSync(os.Stdout)

	baseCore := zapcore.NewCore(
		encoder,
		syncer,
		level,
	)
	if !prettyLogging {
		baseCore = attachBaseFields(baseCore)
	}

	return baseCore
}

func newZapLogger(debug bool, cores ...zapcore.Core) *zap.Logger {
	var zapOpts []zap.Option

	if debug {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	zapOpts = append(zapOpts, zap.AddStacktrace(zap.ErrorLevel))

	zapTee := zapcore.NewTee(cores...)
	zapLogger := zap.New(zapTee, zapOpts...)

	return zapLogger
}

func ZapLogLevelFromString(logLevel string) (zapcore.Level, error) {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return zap.DebugLevel, nil
	case "INFO":
		return zap.InfoLevel, nil
	case "WARNING":
		return zap.WarnLevel, nil
	case "ERROR":
		return zap.ErrorLevel, nil
	case "FATAL":
		return zap.FatalLevel, nil
	case "PANIC":
		return zap.PanicLevel, nil
	default:
		return -1, fmt.Errorf("unknown log level: %s", logLevel)
	}
}

func WithRequestID(reqID string) zap.Field {
	return zap.String(requestIDField, reqID)
}
