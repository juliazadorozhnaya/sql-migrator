package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger интерфейс для логирования
type Logger interface {
	Fatal(msg string, v ...interface{})
	Error(msg string, v ...interface{})
	Warn(msg string, v ...interface{})
	Info(msg string, v ...interface{})
	Debug(msg string, v ...interface{})
}

// Config интерфейс для конфигурации логгера
type Config interface {
	GetLevel() string
}

// logger реализация интерфейса Logger
type logger struct {
}

func New() Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return &logger{}
}

// getLevel возвращает уровень логирования, соответствующий строке
func getLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "fatal":
		return zerolog.FatalLevel
	case "error":
		return zerolog.ErrorLevel
	case "warn":
		return zerolog.WarnLevel
	case "info":
		return zerolog.InfoLevel
	case "debug":
		return zerolog.DebugLevel
	default:
		return zerolog.InfoLevel
	}
}

// Fatal логирует сообщение уровня Fatal
func (l *logger) Fatal(msg string, v ...interface{}) {
	log.Fatal().Msgf(msg, v...)
}

// Error логирует сообщение уровня Error
func (l *logger) Error(msg string, v ...interface{}) {
	log.Error().Msgf(msg, v...)
}

// Warn логирует сообщение уровня Warn
func (l *logger) Warn(msg string, v ...interface{}) {
	log.Warn().Msgf(msg, v...)
}

// Info логирует сообщение уровня Info
func (l *logger) Info(msg string, v ...interface{}) {
	log.Info().Msgf(msg, v...)
}

// Debug логирует сообщение уровня Debug
func (l *logger) Debug(msg string, v ...interface{}) {
	log.Debug().Msgf(msg, v...)
}
