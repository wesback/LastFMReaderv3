package logging

import (
	"fmt"
	"regexp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with secret redaction capability
type Logger struct {
	*zap.Logger
	secretRedactor func(string) string
}

// New creates a new Logger with the specified log level
func New(levelStr string) (*Logger, error) {
	var level zapcore.Level
	switch levelStr {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.DisableCaller = false
	config.DisableStacktrace = false
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Logger{
		Logger:         zapLogger,
		secretRedactor: redactSecret,
	}, nil
}

// redactSecret masks sensitive values, showing only last 4 characters
func redactSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return fmt.Sprintf("****%s", s[len(s)-4:])
}

// WithRedacted returns a field with the value redacted for logging
func (l *Logger) WithRedacted(key, value string) zap.Field {
	return zap.String(key, l.secretRedactor(value))
}

// InfoEvent logs an info-level event with structured fields
func (l *Logger) InfoEvent(event string, fields ...zap.Field) {
	l.Info(event, fields...)
}

// DebugEvent logs a debug-level event with structured fields
func (l *Logger) DebugEvent(event string, fields ...zap.Field) {
	l.Debug(event, fields...)
}

// WarnEvent logs a warn-level event with structured fields
func (l *Logger) WarnEvent(event string, fields ...zap.Field) {
	l.Warn(event, fields...)
}

// ErrorEvent logs an error-level event with structured fields
func (l *Logger) ErrorEvent(event string, fields ...zap.Field) {
	l.Error(event, fields...)
}

// RedactSecrets removes common secret patterns from strings for logging
// Patterns: API keys, SAS tokens, connection strings, Azure credentials
func RedactSecrets(input string) string {
	// Mask API keys pattern (alphanumeric, 10+ chars between api_key= or similar)
	apiKeyRegex := regexp.MustCompile(`(?i)(api[_-]?key|key)[=:\s]+([a-zA-Z0-9]{10,})`)
	input = apiKeyRegex.ReplaceAllString(input, `$1=****`)

	// Mask Azure SAS token pattern (sv=...&sig=...)
	sasRegex := regexp.MustCompile(`(?i)(sig|sas[_-]?token|se|sp)[=:\s]+([^\s&,;]+)`)
	input = sasRegex.ReplaceAllString(input, `$1=****`)

	// Mask Azure connection string patterns (AccountKey=..., SharedAccessKey=...)
	connStrRegex := regexp.MustCompile(`(?i)(AccountKey|SharedAccessKey|AccountName)=([^;]+)`)
	input = connStrRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := regexp.MustCompile(`=`).Split(match, 2)
		if len(parts) == 2 && len(parts[1]) > 4 {
			return parts[0] + "=****" + parts[1][len(parts[1])-4:]
		}
		return parts[0] + "=****"
	})

	// Mask full Azure Storage connection strings
	azureConnRegex := regexp.MustCompile(`DefaultEndpointsProtocol=https;AccountName=[^;]+;AccountKey=[^;]+`)
	input = azureConnRegex.ReplaceAllString(input, `DefaultEndpointsProtocol=https;AccountName=****;AccountKey=****`)

	return input
}
