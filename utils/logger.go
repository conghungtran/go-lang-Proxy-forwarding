package utils

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger() error {
    config := zap.NewProductionConfig()
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    var err error
    Logger, err = config.Build()
    if err != nil {
        return err
    }
    
    return nil
}

func GetLogger() *zap.Logger {
    return Logger
}