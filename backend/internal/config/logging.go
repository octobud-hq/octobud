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

// NewConsoleLoggerWithFile creates a human-readable console logger that writes to both
// stdout and an optional file writer. If fileWriter is nil, it only writes to stdout.
// Output is colored and formatted for easy reading in a terminal.
func NewConsoleLoggerWithFile(fileWriter io.Writer) *zap.Logger {
	// Create encoder config for human-readable output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "",              // Omit caller for cleaner output
		FunctionKey:    zapcore.OmitKey, // Omit function name
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder, // Colored levels
		EncodeTime:     formatTimestamp,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Create sync writers - always write to stdout, optionally also to file
	writeSyncer := zapcore.AddSync(os.Stdout)
	if fileWriter != nil {
		// Write to both stdout and file
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(fileWriter),
		)
	}

	// Create core with info level (skip debug for cleaner output in desktop mode)
	core := zapcore.NewCore(
		consoleEncoder,
		writeSyncer,
		zapcore.InfoLevel,
	)

	return zap.New(core)
}

// NewDebugConsoleLogger creates a verbose console logger with debug level enabled.
// Useful for development and debugging.
func NewDebugConsoleLogger() *zap.Logger {
	return NewDebugConsoleLoggerWithFile(nil)
}

// NewDebugConsoleLoggerWithFile creates a verbose console logger with debug level enabled
// that writes to both stdout and an optional file writer. If fileWriter is nil, it only writes to stdout.
// Useful for development and debugging.
func NewDebugConsoleLoggerWithFile(fileWriter io.Writer) *zap.Logger {
	// Create encoder config for human-readable output with more details
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     formatTimestamp,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Create sync writers - always write to stdout, optionally also to file
	writeSyncer := zapcore.AddSync(os.Stdout)
	if fileWriter != nil {
		// Write to both stdout and file
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(fileWriter),
		)
	}

	// Create core with debug level
	core := zapcore.NewCore(
		consoleEncoder,
		writeSyncer,
		zapcore.DebugLevel,
	)

	return zap.New(core, zap.AddCaller())
}
