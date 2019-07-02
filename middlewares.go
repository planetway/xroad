package xroad

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/pkg/errors"
)

type SOAPMiddleware func(SOAPHandler) SOAPHandler

func ErrorToSOAPFault(next SOAPHandler) SOAPHandler {
	return SOAPHandlerFunc(func(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
		if err := next.ServeSOAP(w, r, e); err != nil {
			if cause, ok := errors.Cause(err).(SOAPFault); ok {
				res := e.NewResponseEnvelope(SOAPFaultBody{
					Fault: cause,
				})
				Log.Info("fault", cause)
				WriteSoap(500, res, w)
				return nil
			}
			Log.Error("error", WrapError(err))
			res := e.NewResponseEnvelope(SOAPFaultBody{
				Fault: SOAPFault{
					Code:   "Server",
					String: "Internal Server Error",
				},
			})
			WriteSoap(500, res, w)
			return nil
		}
		return nil
	})
}

type verboseWriter struct {
	http.ResponseWriter
	io.Writer
}

func NewVerboseResponseWriter(w http.ResponseWriter, ws ...io.Writer) http.ResponseWriter {
	writers := []io.Writer{w}
	writers = append(writers, ws...)
	return verboseWriter{
		ResponseWriter: w,
		Writer:         io.MultiWriter(writers...),
	}
}

func (w verboseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w verboseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w verboseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w verboseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func DumpResponse(next SOAPHandler) SOAPHandler {
	return SOAPHandlerFunc(func(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
		fmt.Fprintf(os.Stderr, "\nout:\n")
		w = NewVerboseResponseWriter(w, os.Stderr)
		return WrapError(next.ServeSOAP(w, r, e))
	})
}

func SOAPHeaderLog(l Logger) func(SOAPHandler) SOAPHandler {
	return func(next SOAPHandler) SOAPHandler {
		return SOAPHandlerFunc(func(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
			l.Info("header", e.Header)
			return WrapError(next.ServeSOAP(w, r, e))
		})
	}
}

func RecoverSOAP(next SOAPHandler) SOAPHandler {
	return SOAPHandlerFunc(func(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) (err error) {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = errors.New("panic")
				}
			}
			if err != nil {
				stack := debug.Stack()
				Log.Error("error", err, "stack", string(stack))
			}
		}()
		return next.ServeSOAP(w, r, e)
	})
}
