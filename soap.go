package xroad

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type SOAPHandler func(http.ResponseWriter, SOAPEnvelope) error

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

func (m *Mux) serveSoap2(w http.ResponseWriter, e SOAPEnvelope) error {
	if h, ok := m.handlers[e.Header.Service.ServiceCode]; ok {
		return WrapError(h(w, e))
	}
	// fallback to "*"
	if h, ok := m.handlers["*"]; ok {
		return WrapError(h(w, e))
	}
	return SOAPFault{
		Code:   "soap:Server",
		String: "Service not found",
	}
}

func (m *Mux) serveSoap(w http.ResponseWriter, e SOAPEnvelope) error {
	next := m.serveSoap2
	for _, middleware := range m.Middlewares {
		next = middleware(next)
	}
	return WrapError(next(w, e))
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	var e SOAPEnvelope
	e.Body = m.NewBody()
	if err := Decode(contentType, r.Body, &e); err != nil {
		return WrapError(err)
	}
	return WrapError(m.serveSoap(w, e))
}

func Decode(contentType string, r io.Reader, envelope *SOAPEnvelope) error {
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
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(status)

		enc := xml.NewEncoder(w)
		return WrapError(enc.Encode(e))
	} else {
		e.XOP.SOAPEnvelope = e
		_, err := e.XOP.WriteResponse(w)
		return WrapError(err)
	}
}
