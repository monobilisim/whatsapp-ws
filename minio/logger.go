package minio

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Panic(args ...interface{})
	Fatal(args ...interface{})
	DebugWithFields(fields map[string]interface{}, args ...interface{})
	InfoWithFields(fields map[string]interface{}, args ...interface{})
	WarnWithFields(fields map[string]interface{}, args ...interface{})
	ErrorWithFields(fields map[string]interface{}, args ...interface{})
	PanicWithFields(fields map[string]interface{}, args ...interface{})
	FatalWithFields(fields map[string]interface{}, args ...interface{})
}
