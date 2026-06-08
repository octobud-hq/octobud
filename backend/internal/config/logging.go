// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package config provides logging configuration for the application.
package config

import (
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// formatTimestamp returns a human-readable timestamp encoder for zap.
func formatTimestamp(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

// NewConsoleLogger creates a human-readable console logger suitable for desktop/CLI applications.
// Output is colored and formatted for easy reading in a terminal.
func NewConsoleLogger() *zap.Logger {
	return NewConsoleLoggerWithFile(nil)
}

// NewConsoleLoggerWithFile creates a logger that writes human-readable console
// output to stdout and structured JSON to an optional file writer. The JSON file
// format lets the in-app log viewer parse entries reliably; if fileWriter is
// nil, only stdout is used.
func NewConsoleLoggerWithFile(fileWriter io.Writer) *zap.Logger {
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	if fileWriter == nil {
		return zap.New(consoleCore)
	}

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(jsonEncoderConfig()),
		zapcore.AddSync(fileWriter),
		zapcore.InfoLevel,
	)

	return zap.New(zapcore.NewTee(consoleCore, fileCore))
}

// consoleEncoderConfig is the human-readable config used for stdout.
func consoleEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     formatTimestamp,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// jsonEncoderConfig is used for the on-disk log so the diagnostics viewer can
// parse entries one-per-line. Uses ISO8601 timestamps and lowercase level names
// for stable downstream consumption.
func jsonEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// NewDebugConsoleLogger creates a verbose console logger with debug level enabled.
// Useful for development and debugging.
func NewDebugConsoleLogger() *zap.Logger {
	return NewDebugConsoleLoggerWithFile(nil)
}

// NewDebugConsoleLoggerWithFile creates a verbose debug-level logger that writes
// console-formatted output to stdout and JSON to an optional file writer.
// Useful for development and debugging.
func NewDebugConsoleLoggerWithFile(fileWriter io.Writer) *zap.Logger {
	debugConsoleConfig := consoleEncoderConfig()
	debugConsoleConfig.CallerKey = "caller"

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(debugConsoleConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	if fileWriter == nil {
		return zap.New(consoleCore, zap.AddCaller())
	}

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(jsonEncoderConfig()),
		zapcore.AddSync(fileWriter),
		zapcore.DebugLevel,
	)

	return zap.New(zapcore.NewTee(consoleCore, fileCore), zap.AddCaller())
}
