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

type SOAPClient struct {
	http.Client
}

type Client struct {
	SOAPClient
	IdGenerator // override if you want your own Id generator other than uuid.NewV4
	Url         string
	// header is reused when Send() ing multiple requests using this Client,
	// so mutating Header will effect all future requests.
	// Use Set* funcs to set the header while keeping the original client unchanged.
	header SOAPHeader
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
			u, err := uuid.NewV4()
			return u.String(), WrapError(err)
		},
		Url:    url,
		header: h,
	}
}

// Although the response.Body is already read in Send() to parse the response into SOAP,
// it is the caller's responsibility to close response.Body.
// The resEnvelope might include XOP files, and those should be read until EOF
// before closing the response.Body.
func (c SOAPClient) Send(url string, h SOAPHeader, body interface{}, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := c.NewRequest(url, h, body)
	if err != nil {
		return nil, WrapError(err)
	}

	return c.doAndDecode(req, resEnvelope)
}

func (c SOAPClient) SendXOP(url string, h SOAPHeader, body FileIncluder, r io.Reader, filename string, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	req, err := NewXOPRequestFromReader(url, h, body, r, filename)
	if err != nil {
		return nil, WrapError(err)
	}

	return c.doAndDecode(req, resEnvelope)
}

func (c SOAPClient) doAndDecode(req *http.Request, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	res, err := c.Do(req)
	if err != nil {
		return nil, WrapError(err)
	}

	if err := DecodeResponse(res, resEnvelope); err != nil {
		return nil, WrapError(err)
	}

	return res, WrapError(err)
}

func (c SOAPClient) Do(req *http.Request) (*http.Response, error) {
	res, err := c.Client.Do(req)

	return res, WrapError(err)
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

func (c Client) Send(body interface{}, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	id, err := c.IdGenerator()
	if err != nil {
		return nil, WrapError(err)
	}
	h := c.header
	h.Id = id
	return c.SOAPClient.Send(c.Url, h, body, resEnvelope)
}

func (c Client) SendXOP(body FileIncluder, r io.Reader, filename string, resEnvelope *SOAPEnvelope) (*http.Response, error) {
	id, err := c.IdGenerator()
	if err != nil {
		return nil, WrapError(err)
	}
	h := c.header
	h.Id = id
	return c.SOAPClient.SendXOP(c.Url, h, body, r, filename, resEnvelope)
}

// SetUserId mutates UserId in SOAP header.
// Make a copy of Client before calling this if you're reusing the client.
func (c *Client) SetUserId(u string) *Client {
	c.header.UserId = u
	return c
}

// SetServiceCode mutates serviceCode and optionally serviceVersion in SOAP header.
// Make a copy of Client before calling this if you're reusing the client.
func (c *Client) SetServiceCode(serviceCode ...string) *Client {
	version := c.header.Service.ServiceVersion
	if len(serviceCode) > 1 {
		version = serviceCode[1]
	}
	c.header.Service.ServiceVersion = version
	c.header.Service.ServiceCode = serviceCode[0]
	return c
}

// SetProvider mutates Service's Subsystem part of SOAP header.
// Make a copy of Client before calling this if you're reusing the client.
func (c *Client) SetProvider(s XroadClient) *Client {
	cl := c.header.Service.XroadClient
	cl.XRoadInstance = s.XRoadInstance
	cl.MemberClass = s.MemberClass
	cl.MemberCode = s.MemberCode
	cl.SubsystemCode = s.SubsystemCode
	c.header.Service.XroadClient = cl
	return c
}

// SetHeader mutates c.header.
// Make a copy of Client before calling this if you're reusing the client.
func (c *Client) SetHeader(f func(SOAPHeader) SOAPHeader) *Client {
	h := f(c.header)
	c.header = h
	return c
}
