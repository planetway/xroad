package xroad

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
)

var (
	UserAgent = "xroad.go"
)

type IdGenerator func() (string, error)

type Client struct {
	http.Client
	IdGenerator // override if you want your own Id generator other than uuid.NewV4
}

func NewClient() Client {
	return Client{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		IdGenerator: func() (string, error) {
			u, err := uuid.NewV4()
			return u.String(), WrapError(err)
		},
	}
}

// Although the response.Body is already read in Send() to parse the response into SOAP,
// it is the caller's responsibility to close response.Body.
// The resEnvelope might include XOP files, and those should be read until EOF
// before closing the response.Body.
func (c Client) Send(url string, h SOAPHeader, body interface{}, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := c.NewRequest(url, h, body)
	if err != nil {
		return nil, WrapError(err)
	}

	return c.doAndDecode(req, resEnvelope)
}

func (c Client) SendXOP(url string, h SOAPHeader, body FileIncluder, r io.Reader, filename string, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := NewXOPRequestFromReader(url, h, body, r, filename)
	if err != nil {
		return nil, WrapError(err)
	}

	return c.doAndDecode(req, resEnvelope)
}

func (c Client) doAndDecode(req *http.Request, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	res, err := c.Do(req)
	if err != nil {
		return nil, WrapError(err)
	}

	if err := DecodeResponse(res, resEnvelope); err != nil {
		return nil, WrapError(err)
	}

	return res, WrapError(err)
}

func (c Client) Do(req *http.Request) (*http.Response, error) {
	res, err := c.Client.Do(req)

	return res, WrapError(err)
}

func (c Client) NewRequest(url string, header SOAPHeader, body interface{}) (*http.Request, error) {
	id, err := c.IdGenerator()
	if err != nil {
		return nil, WrapError(err)
	}
	header.Id = id
	e := NewEnvelope(header, body)

	b, err := xml.Marshal(e)
	if err != nil {
		return nil, WrapError(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return nil, WrapError(err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("User-Agent", UserAgent)

	return req, nil
}
