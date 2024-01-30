package logger

import (
	"github.com/mdesson/chatcord/util"
	"log/slog"
	"os"
	"runtime/debug"
)

type Logger struct {
	l *slog.Logger
}

func New(level slog.Level) *Logger {
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	return &Logger{l: l}
}

func (l *Logger) Info(msg string, args ...any) {
	args = append(args, "function", util.FunctionName(2))
	l.l.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	args = append(args, "function", util.FunctionName(2))
	l.l.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	args = append(args, "function", util.FunctionName(2))
	args = append(args, "stack_trace", string(debug.Stack()))
	l.l.Error(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	args = append(args, "function", util.FunctionName(2))
	l.l.Debug(msg, args...)
}
