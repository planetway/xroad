package xroad

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	UserAgent = "xroad.go"
)

type IdGenerator func() (string, error)

type SOAPClient struct {
	http.Client
}

type Client struct {
	SOAPClient
	IdGenerator // override if you want your own Id generator other than uuid.NewV4
	Url         string
	baseHeader  SOAPHeader
}

func NewSOAPClient() SOAPClient {
	return SOAPClient{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewClient(url string, h SOAPHeader) Client {
	return Client{
		SOAPClient: NewSOAPClient(),
		IdGenerator: func() (string, error) {
			uuid, err := uuid.NewUUID()
			return uuid.String(), WrapError(err)
		},
		Url:        url,
		baseHeader: h,
	}
}

func (c Client) CloneHeader() SOAPHeader {
	ret := c.baseHeader
	// copy values, not addresses
	if ret.Service != nil {
		s := *ret.Service
		ret.Service = &s
	}
	if ret.CentralService != nil {
		s := *ret.CentralService
		ret.CentralService = &s
	}
	return ret
}

func (c SOAPClient) NewRequest(url string, header SOAPHeader, body interface{}) (*http.Request, error) {
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

func (c Client) NewRequest(header SOAPHeader, body interface{}) (*http.Request, error) {
	id, err := c.IdGenerator()
	if err != nil {
		return nil, WrapError(err)
	}
	header.Id = id
	header.fillDefaults()

	req, err := c.SOAPClient.NewRequest(c.Url, header, body)
	return req, WrapError(err)
}

// Although the response.Body is already read in Send() to parse the response into SOAP,
// it is the caller's responsibility to close response.Body.
// The resEnvelope might include XOP files, and those should be read until EOF
// before closing the response.Body.
func (c Client) Send(header SOAPHeader, body interface{}, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := c.NewRequest(header, body)
	if err != nil {
		return nil, WrapError(err)
	}
	res, err := c.doAndDecode(req, resEnvelope)
	return res, WrapError(err)
}

// Although the response.Body is already read in Send() to parse the response into SOAP,
// it is the caller's responsibility to close response.Body.
// The resEnvelope might include XOP files, and those should be read until EOF
// before closing the response.Body.
func (c Client) SendXOP(header SOAPHeader, body FileIncluder, r io.Reader, filename string, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := NewXOPRequestFromReader(c.Url, header, body, r, filename)
	if err != nil {
		return nil, WrapError(err)
	}
	res, err := c.doAndDecode(req, resEnvelope)
	return res, WrapError(err)
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
