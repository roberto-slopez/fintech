package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wrapper para zerolog con funcionalidades adicionales
type Logger struct {
	zerolog.Logger
}

// NewLogger crea un nuevo logger configurado
func NewLogger() *Logger {
	return NewLoggerWithConfig("info", "console", "stdout", "")
}

// NewLoggerWithConfig crea un logger con configuración específica
func NewLoggerWithConfig(level, format, output, filePath string) *Logger {
	var writer io.Writer
	
	switch output {
	case "file":
		if filePath != "" {
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				writer = file
			} else {
				writer = os.Stdout
			}
		} else {
			writer = os.Stdout
		}
	default:
		writer = os.Stdout
	}
	
	// Formato de salida
	if format == "console" {
		writer = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
	}
	
	// Nivel de log
	var logLevel zerolog.Level
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}
	
	logger := zerolog.New(writer).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()
	
	return &Logger{logger}
}

// WithRequestID agrega un request ID al logger
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{l.With().Str("request_id", requestID).Logger()}
}

// WithUserID agrega un user ID al logger
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{l.With().Str("user_id", userID).Logger()}
}

// WithApplicationID agrega un application ID al logger
func (l *Logger) WithApplicationID(applicationID string) *Logger {
	return &Logger{l.With().Str("application_id", applicationID).Logger()}
}

// WithCountry agrega el código de país al logger
func (l *Logger) WithCountry(countryCode string) *Logger {
	return &Logger{l.With().Str("country", countryCode).Logger()}
}

// WithJobID agrega un job ID al logger
func (l *Logger) WithJobID(jobID string) *Logger {
	return &Logger{l.With().Str("job_id", jobID).Logger()}
}

// WithWorkerID agrega un worker ID al logger
func (l *Logger) WithWorkerID(workerID string) *Logger {
	return &Logger{l.With().Str("worker_id", workerID).Logger()}
}

// WithError agrega un error al logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.With().Err(err).Logger()}
}

// WithFields agrega múltiples campos al logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{ctx.Logger()}
}

