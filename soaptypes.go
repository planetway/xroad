package xroad

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Value only struct
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  SOAPHeader  `xml:""`
	Body    interface{} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	XOP     *XOP        `xml:"-"`
}

func NewEnvelope(h SOAPHeader, b interface{}) SOAPEnvelope {
	return SOAPEnvelope{
		Header: h,
		Body:   b,
	}
}

func (x SOAPEnvelope) NewResponseEnvelope(body interface{}) SOAPEnvelope {
	res := x
	res.Body = body
	res.XOP = nil
	return res
}

func (x SOAPEnvelope) String() string {
	return fmt.Sprintf("header: [%s], body: [%s]", x.Header, x.Body)
}

type SOAPHeader struct {
	XMLName         xml.Name     `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header" json:"-"`
	ProtocolVersion string       `xml:"http://x-road.eu/xsd/xroad.xsd protocolVersion" json:"protocolVersion"`
	Id              string       `xml:"http://x-road.eu/xsd/xroad.xsd id" json:"id"`
	UserId          string       `xml:"http://x-road.eu/xsd/xroad.xsd userId" json:"userId"`
	TargetUserId    string       `xml:"http://x-road.eu/xsd/xroad.xsd targetUserId,omitempty" json:"targetUserId"`
	Issue           string       `xml:"http://x-road.eu/xsd/xroad.xsd issue,omitempty" json:"issue"`
	Service         XroadService `xml:"service" json:"service" mapstructure:"service"`
	Client          XroadClient  `xml:"client" json:"client" mapstructure:"client"`
}

func (x SOAPHeader) String() string {
	return fmt.Sprintf("v: %s, id: %s, userId: %s, targetUserId: %s, issue: %s, service: [%s], client: [%s]",
		x.ProtocolVersion, x.Id, x.UserId, x.TargetUserId, x.Issue, x.Service, x.Client)
}

type XroadService struct {
	XroadClient    `mapstructure:",squash"`
	XMLName        xml.Name `xml:"http://x-road.eu/xsd/xroad.xsd service" json:"-"`
	ServiceCode    string   `xml:"http://x-road.eu/xsd/identifiers serviceCode" json:"serviceCode"`
	ServiceVersion string   `xml:"http://x-road.eu/xsd/identifiers serviceVersion" json:"serviceVersion"`
}

// Create a new XroadService from service FQDN.
// Reading code like FiVRKSignCertificateProfileInfo.java , we assume all the parts don't include a '/'
func NewXroadService(fqdn string) (*XroadService, error) {
	parts := strings.Split(fqdn, ".")
	if len(parts) != 6 {
		return nil, WrapError(errors.New("invalid service fqdn"))
	}
	return &XroadService{
		XroadClient: XroadClient{
			ObjectType:    "",
			XRoadInstance: parts[0],
			MemberClass:   parts[1],
			MemberCode:    parts[2],
			SubsystemCode: parts[3],
		},
		ServiceCode:    parts[4],
		ServiceVersion: parts[5],
	}, nil
}

func (x XroadService) Equal(y XroadService) bool {
	if x.XroadClient.Equal(y.XroadClient) &&
		x.ServiceCode == y.ServiceCode &&
		x.ServiceVersion == y.ServiceVersion {
		return true
	}
	return false
}

func (x XroadService) Fqdn() string {
	return fmt.Sprintf("%s.%s.%s", x.XroadClient.Fqdn(), x.ServiceCode, x.ServiceVersion)
}

func (x XroadService) String() string {
	return x.Fqdn()
}

type XroadClient struct {
	XMLName       xml.Name `xml:"http://x-road.eu/xsd/xroad.xsd client" json:"-"`
	ObjectType    string   `xml:"http://x-road.eu/xsd/identifiers objectType,attr" json:"objectType"`
	XRoadInstance string   `xml:"http://x-road.eu/xsd/identifiers xRoadInstance" json:"xRoadInstance"`
	MemberClass   string   `xml:"http://x-road.eu/xsd/identifiers memberClass" json:"memberClass"`
	MemberCode    string   `xml:"http://x-road.eu/xsd/identifiers memberCode" json:"memberCode"`
	SubsystemCode string   `xml:"http://x-road.eu/xsd/identifiers subsystemCode" json:"subsystemCode"`
}

// Create a new XroadClient from subsystem FQDN.
// Reading code like FiVRKSignCertificateProfileInfo.java , we assume all the parts don't include a '.'
// ex: JP-TEST.COM.12973914.librarian
func NewXroadClient(fqdn string) (*XroadClient, error) {
	parts := strings.Split(fqdn, ".")
	if len(parts) != 4 {
		return nil, WrapError(errors.New("invalid client fqdn"))
	}
	return &XroadClient{
		ObjectType:    "",
		XRoadInstance: parts[0],
		MemberClass:   parts[1],
		MemberCode:    parts[2],
		SubsystemCode: parts[3],
	}, nil
}

func (x XroadClient) SameMember(y XroadClient) bool {
	if x.XRoadInstance == y.XRoadInstance &&
		x.MemberClass == y.MemberClass &&
		x.MemberCode == y.MemberCode {
		return true
	}
	return false
}

func (x XroadClient) Equal(y XroadClient) bool {
	if x.XRoadInstance == y.XRoadInstance &&
		x.MemberClass == y.MemberClass &&
		x.MemberCode == y.MemberCode &&
		x.SubsystemCode == y.SubsystemCode {
		return true
	}
	return false
}

func (x XroadClient) Fqdn() string {
	return fmt.Sprintf("%s.%s.%s.%s", x.XRoadInstance, x.MemberClass, x.MemberCode, x.SubsystemCode)
}

func (x XroadClient) String() string {
	return x.Fqdn()
}

type SOAPFaultBody struct {
	XMLName xml.Name  `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Fault   SOAPFault `xml:""`
}

type SOAPFault struct {
	XMLName xml.Name         `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string           `xml:"faultcode,omitempty"`
	String  string           `xml:"faultstring,omitempty"`
	Actor   string           `xml:"faultactor,omitempty"`
	Detail  *SOAPFaultDetail `xml:""`
	Cause   error            `xml:"-"` // only used for logging
}

func (s SOAPFault) Error() string {
	cause := "-"
	if s.Cause != nil {
		cause = s.Cause.Error()
	}
	return fmt.Sprintf("faultcode: %s, faultstring: %s, faultactor: %s, %s, cause: %s", s.Code, s.String, s.Actor, s.Detail, cause)
}

// reuse http status codes here
func NewSOAPFault(code int, detail string) SOAPFault {
	return SOAPFault{
		Code:   "soap:Server",
		String: http.StatusText(code),
		Detail: &SOAPFaultDetail{
			FaultDetail: detail,
		},
	}
}

func NewSOAPFaultWithCause(code int, detail string, cause error) SOAPFault {
	return SOAPFault{
		Code:   "soap:Server",
		String: http.StatusText(code),
		Detail: &SOAPFaultDetail{
			FaultDetail: detail,
		},
		Cause: cause,
	}
}

type SOAPFaultDetail struct {
	XMLName     xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ detail"`
	FaultDetail string   `xml:"faultDetail,omitempty"`
}

func (s SOAPFaultDetail) Error() string {
	return fmt.Sprintf("faultDetail: %s", s.FaultDetail)
}

type XOPInclude struct {
	XMLName xml.Name `xml:"http://www.w3.org/2004/08/xop/include Include"`
	Href    string   `xml:"href,attr"`
}
