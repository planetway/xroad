package xroad

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"

	accesslog "github.com/mash/go-accesslog"
	"github.com/pkg/errors"
)

type HTTPHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request) error
}

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return h(w, r)
}

type HTTPError struct {
	Code  int
	Str   string
	Cause error
}

func NewHTTPError(i int) error {
	return HTTPError{
		Code:  i,
		Str:   http.StatusText(i),
		Cause: nil,
	}
}

func (e HTTPError) Error() string {
	return e.Str
}

func ErrorTo500(h HTTPHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h.ServeHTTP(w, r); err != nil {
			cause := errors.Cause(err)
			switch e := cause.(type) {
			case HTTPError:
				http.Error(w, e.Str, e.Code)
			default:
				Log.Error("error", err)
				http.Error(w, "Internal Server Error", 500)
			}
		}
	})
}

func AccessLog(logger Logger) func(http.Handler) http.Handler {
	return accesslog.NewLoggingMiddleware(accessLogger{
		logger: logger,
	})
}

type accessLogger struct {
	logger Logger
}

func (l accessLogger) Log(record accesslog.LogRecord) {
	ip := record.RequestHeader.Get("x-real-ip")
	if ip == "" {
		ip = record.RequestHeader.Get("x-forwarded-for")
	}
	l.logger.Info("ip", ip,
		"host", record.Host,
		"method", record.Method,
		"uri", record.Uri,
		"status", record.Status,
		"size", record.Size,
		"ua", record.RequestHeader.Get("user-agent"),
		"reqtime", fmt.Sprintf("%.3f", record.ElapsedTime.Seconds()))
}

func DumpRequest(h HTTPHandler) HTTPHandler {
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprintf(os.Stderr, "\nin:\n")
		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, os.Stderr))
		return WrapError(h.ServeHTTP(w, r))
	})
}

func RecoverHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					var err error = nil
					switch x := r.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = x
					default:
						err = errors.New("panic")
					}
					stack := debug.Stack()
					Log.Error("error", err, "stack", string(stack))

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
