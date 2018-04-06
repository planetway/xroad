package xroad

// the same as go-kit/kit/log/Logger but without importing it
type Logger interface {
	Log(keyvals ...interface{}) error
}

var (
	Log Logger = nopLog{}
)

type nopLog struct{}

func (_ nopLog) Log(keyvals ...interface{}) error {
	return nil
}
