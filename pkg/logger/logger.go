package logger

type Logger interface {
	Info(input string, fields ...any)
	Error(input string, fields ...any)
	Debug(input string, fields ...any)
	Warn(input string, fields ...any)
	With(args ...any) Logger
}
