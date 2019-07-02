package xroad

// we need levels
type Logger interface {
	Debug(keyvals ...interface{}) error
	Info(keyvals ...interface{}) error
	Error(keyvals ...interface{}) error
}

var (
	Log Logger = nopLog{}
)

type nopLog struct{}

func (_ nopLog) Debug(keyvals ...interface{}) error { return nil }
func (_ nopLog) Info(keyvals ...interface{}) error  { return nil }
func (_ nopLog) Error(keyvals ...interface{}) error { return nil }
