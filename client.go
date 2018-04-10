package xroad

import (
	"bytes"
	"encoding/xml"
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

func (c Client) Send(url string, h SOAPHeader, body interface{}) (*http.Response, error) {
	req, err := c.NewRequest(url, h, body)
	if err != nil {
		return nil, WrapError(err)
	}

	res, err := c.Do(req)

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

func (c Client) NewXOPRequest(url string, header SOAPHeader, xop XOP) (*http.Request, error) {
	buf := &bytes.Buffer{}
	_, err := xop.WriteTo(buf)
	if err != nil {
		return nil, WrapError(err)
	}
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, WrapError(err)
	}
	req.Header.Set("Content-Type", xop.ContentType())
	req.Header.Set("User-Agent", UserAgent)

	return req, nil
}
