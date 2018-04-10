package xroad

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

type SOAPHandlerFunc func(http.ResponseWriter, *http.Request, SOAPEnvelope) error

func (f SOAPHandlerFunc) ServeSOAP(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
	return f(w, r, e)
}

type SOAPHandler interface {
	ServeSOAP(http.ResponseWriter, *http.Request, SOAPEnvelope) error
}

type Mux struct {
	handlers    map[string]SOAPHandler
	Middlewares []SOAPMiddleware
	body        interface{}
}

func VerboseMiddlewares() []SOAPMiddleware {
	return []SOAPMiddleware{
		ErrorToSOAPFault,
		DumpResponse,
		RecoverSOAP,
	}
}

func NewMux(body interface{}) *Mux {
	return &Mux{
		handlers: make(map[string]SOAPHandler),
		Middlewares: []SOAPMiddleware{
			ErrorToSOAPFault,
			SOAPHeaderLog(Log),
			RecoverSOAP,
		},
		body: body,
	}
}

func (m *Mux) Handle(pattern string, h SOAPHandler) {
	m.handlers[pattern] = h
}

func (m *Mux) serveSoap2(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
	if h, ok := m.handlers[e.Header.Service.ServiceCode]; ok {
		return WrapError(h.ServeSOAP(w, r, e))
	}
	// fallback to "*"
	if h, ok := m.handlers["*"]; ok {
		return WrapError(h.ServeSOAP(w, r, e))
	}
	return SOAPFault{
		Code:   "soap:Server",
		String: "Service not found",
	}
}

func (m *Mux) serveSoap(w http.ResponseWriter, r *http.Request, e SOAPEnvelope) error {
	var next SOAPHandler
	next = SOAPHandlerFunc(m.serveSoap2)
	for _, middleware := range m.Middlewares {
		next = middleware(next)
	}
	return WrapError(next.ServeSOAP(w, r, e))
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return nil
	}

	var e SOAPEnvelope
	e.Body = m.NewBody()
	if err := Decode(r, &e); err != nil {
		return WrapError(err)
	}
	return WrapError(m.serveSoap(w, r, e))
}

// Decode parses the request.Body to xroad.SOAPEnvelope or to xroad.XOP
// depending on the Content-Type request header.
// After the body is read, we seek to the start of the request.Body
// future consumers.
func Decode(r *http.Request, envelope *SOAPEnvelope) error {
	contentType := r.Header.Get("Content-Type")

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return WrapError(err)
	}
	body := bytes.NewReader(b)

	if err := DecodeReader(body, contentType, envelope); err != nil {
		return WrapError(err)
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return WrapError(err)
	}
	r.Body = ioutil.NopCloser(body)
	return nil
}

func DecodeReader(r io.Reader, contentType string, envelope *SOAPEnvelope) error {
	if strings.HasPrefix(contentType, "text/xml") {
		// parse SOAP
		dec := xml.NewDecoder(r)
		if err := dec.Decode(envelope); err != nil {
			return WrapError(err)
		}
		return nil
	} else if strings.HasPrefix(contentType, "multipart/") {
		// parse multipart
		xop, err := NewXOPFromReader(contentType, r, envelope)
		if err != nil {
			return WrapError(err)
		}
		envelope.XOP = xop
		return nil
	}
	return WrapError(errors.New("invalid Content-Type"))
}

func (m *Mux) NewBody() interface{} {
	return reflect.New(reflect.TypeOf(m.body)).Interface()
}

func WriteSoap(status int, e SOAPEnvelope, w http.ResponseWriter) error {
	if e.XOP == nil {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(status)

		enc := xml.NewEncoder(w)
		return WrapError(enc.Encode(e))
	} else {
		e.XOP.SOAPEnvelope = e
		_, err := e.XOP.WriteResponse(w)
		return WrapError(err)
	}
}
