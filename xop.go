package xroad

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/satori/go.uuid"
)

type XOP struct {
	SOAPEnvelope SOAPEnvelope
	Boundary     string
	Files        []xopFile
}

type xopFile struct {
	ContentId, Filename string
	File                io.Reader
}

func NewXOP() (XOP, error) {
	u, err := uuid.NewV4()
	if err != nil {
		return XOP{}, WrapError(err)
	}
	return XOP{
		Boundary: fmt.Sprintf("uuid:%s", u.String()),
	}, nil
}

type FileIncluder interface {
	IncludeFile(string)
}

func NewXOPRequestFromReader(url string, header SOAPHeader, body FileIncluder, r io.Reader, filename string) (*http.Request, error) {
	xop, err := NewXOP()
	if err != nil {
		return nil, WrapError(err)
	}
	cid, err := xop.AddFile(filename, r)
	if err != nil {
		return nil, WrapError(err)
	}

	body.IncludeFile(cid)
	xop.SOAPEnvelope = NewEnvelope(header, body)

	return NewXOPRequest(url, header, xop)
}

func NewXOPRequest(url string, header SOAPHeader, xop XOP) (*http.Request, error) {
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

func (x *XOP) AddFile(filename string, r io.Reader) (string, error) {
	u, err := uuid.NewV4()
	if err != nil {
		return "", WrapError(err)
	}
	cid := u.String()
	x.Files = append(x.Files, xopFile{
		ContentId: cid,
		Filename:  filename,
		File:      r,
	})
	return cid, nil
}

func (x *XOP) ContentType() string {
	return fmt.Sprintf(`multipart/related; type="application/xop+xml"; boundary="%s"; start="<root>"; start-info="text/xml"`, x.Boundary)
}

func (x *XOP) WriteResponse(w http.ResponseWriter) (n int64, err error) {
	w.Header().Set("Content-Type", x.ContentType())
	return x.WriteTo(w)
}

func (x *XOP) WriteTo(w io.Writer) (n int64, err error) {
	mw := multipart.NewWriter(w)
	mw.SetBoundary(x.Boundary)

	h1 := make(textproto.MIMEHeader)
	// same as seen in https://github.com/vrk-kpa/X-Road/blob/develop/doc/Protocols/pr-mess_x-road_message_protocol.md#annex-g-example-request-with-mtom-attachment
	h1.Add("Content-Type", `application/xop+xml; charset=UTF-8; type="text/xml"`)
	h1.Add("Content-Transfer-Encoding", "8bit")
	h1.Add("Content-ID", "<root>")
	root, err := mw.CreatePart(h1)
	if err != nil {
		return 0, WrapError(err)
	}

	enc := xml.NewEncoder(root)
	if err := enc.Encode(x.SOAPEnvelope); err != nil {
		return 0, WrapError(err)
	}

	for _, file := range x.Files {
		h2 := make(textproto.MIMEHeader)
		// same as seen in https://github.com/vrk-kpa/X-Road/blob/develop/doc/Protocols/pr-mess_x-road_message_protocol.md#annex-g-example-request-with-mtom-attachment
		h2.Add("Content-Type", fmt.Sprintf("application/octet-stream; name=%s", file.Filename))
		h2.Add("Content-Transfer-Encoding", "base64")
		h2.Add("Content-ID", file.ContentId)
		h2.Add("Content-Disposition", fmt.Sprintf(`attachment;name="%s";filename="%s"`, file.Filename, file.Filename))
		part, err := mw.CreatePart(h2)
		if err != nil {
			return 0, WrapError(err)
		}
		enc := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := io.Copy(enc, file.File); err != nil {
			return 0, WrapError(err)
		}
		enc.Close()
	}
	if err := mw.Close(); err != nil {
		return 0, WrapError(err)
	}

	return 0, nil
}

func NewXOPFromReader(contentType string, r io.Reader, envelope *SOAPEnvelope) (*XOP, error) {
	x := &XOP{}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return x, WrapError(err)
	}
	x.Boundary = params["boundary"]
	if x.Boundary == "" {
		return x, WrapError(fmt.Errorf("boundary not found, Content-Type: %s", contentType))
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return x, WrapError(fmt.Errorf("mediaType does not look like multipart"))
	}

	mr := multipart.NewReader(r, params["boundary"])

	first := true
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return x, nil
		}
		if err != nil {
			return x, WrapError(err)
		}
		if first {
			first = false
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return x, WrapError(err)
			}
			if err := xml.Unmarshal(b, envelope); err != nil {
				return x, WrapError(err)
			}
			x.SOAPEnvelope = *envelope
		} else {
			_, err = x.AddFile(part.FileName(), part)
			if err != nil {
				return nil, WrapError(err)
			}
			// only allow one xopFile
			break
		}
	}
	return x, nil
}
