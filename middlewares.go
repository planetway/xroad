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
	return func(w http.ResponseWriter, e SOAPEnvelope) error {
		if err := next(w, e); err != nil {
			if cause, ok := errors.Cause(err).(SOAPFault); ok {
				res := e.NewResponseEnvelope(SOAPFaultBody{
					Fault: cause,
				})
				Log.Log("fault", cause)
				WriteSoap(500, res, w)
				return nil
			}
			Log.Log("error", WrapError(err))
			res := e.NewResponseEnvelope(SOAPFaultBody{
				Fault: SOAPFault{
					Code:   "soap:Server",
					String: "Internal Server Error",
				},
			})
			WriteSoap(500, res, w)
			return nil
		}
		return nil
	}
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
	return func(w http.ResponseWriter, e SOAPEnvelope) error {
		fmt.Fprintf(os.Stderr, "\nout:\n")
		w = NewVerboseResponseWriter(w, os.Stderr)
		return WrapError(next(w, e))
	}
}

func SOAPHeaderLog(l Logger) func(SOAPHandler) SOAPHandler {
	return func(next SOAPHandler) SOAPHandler {
		return func(w http.ResponseWriter, e SOAPEnvelope) error {
			l.Log("id", e.Header.Id, "userId", e.Header.UserId, "client", e.Header.Client.String(), "service", e.Header.Service.String())
			return WrapError(next(w, e))
		}
	}
}

func RecoverSOAP(next SOAPHandler) SOAPHandler {
	return func(w http.ResponseWriter, e SOAPEnvelope) (err error) {
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
				Log.Log("error", err, "stack", stack)
			}
		}()
		return next(w, e)
	}
}
