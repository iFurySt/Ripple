package logger

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	Filename    string `yaml:"filename"`
	MaxSize     int    `yaml:"max_size"`
	MaxAge      int    `yaml:"max_age"`
	MaxBackups  int    `yaml:"max_backups"`
	LocalTime   bool   `yaml:"local_time"`
	Compress    bool   `yaml:"compress"`
	Console     bool   `yaml:"console"`
	TimeFormat  string `yaml:"time_format"`
	Timezone    string `yaml:"timezone"`
}

func NewLogger(cfg Config) (*zap.Logger, error) {
	// Set default values
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "console"
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = "2006-01-02 15:04:05"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "Local"
	}

	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder(cfg.TimeFormat, cfg.Timezone),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   customCallerEncoder,
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create writers
	var writers []zapcore.WriteSyncer

	// Console writer
	if cfg.Console {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// File writer
	if cfg.Filename != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			LocalTime:  cfg.LocalTime,
			Compress:   cfg.Compress,
		})
		writers = append(writers, fileWriter)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	// Create logger
	logger := zap.New(core, zap.AddCaller())

	return logger, nil
}

func customTimeEncoder(format, timezone string) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		var loc *time.Location
		var err error

		if timezone == "Local" {
			loc = time.Local
		} else {
			loc, err = time.LoadLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
		}

		enc.AppendString(t.In(loc).Format(format))
	}
}

func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// Get the full path and extract relevant parts
	fullPath := caller.FullPath()
	
	// Try to extract the most relevant part of the path
	if strings.Contains(fullPath, "/ripple/") {
		// Find the project root and show path from there
		parts := strings.Split(fullPath, "/ripple/")
		if len(parts) > 1 {
			enc.AppendString(parts[len(parts)-1])
			return
		}
	}
	
	// Fallback to short caller if our custom logic doesn't work
	enc.AppendString(caller.TrimmedPath())
}