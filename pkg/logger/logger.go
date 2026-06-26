package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	L *zap.Logger
	S *zap.SugaredLogger
)

func Init(serviceName string) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(zapcore.InfoLevel),
	)
	L = zap.New(core, zap.Fields(zap.String("service", serviceName)))
	S = L.Sugar()
	log.Printf("[Logger] Initialized for service: %s", serviceName)
}

func Get() *zap.Logger {
	if L == nil {
		Init("unknown")
	}
	return L
}

func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
