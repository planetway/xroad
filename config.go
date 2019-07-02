package xroad

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
)

type ReqConfig struct {
	Url        string     `json:"url" mapstructure:"url"`
	SOAPHeader SOAPHeader `json:"header" yaml:"header" mapstructure:"header"`
}

func LoadConfig(filename string) (*ReqConfig, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, WrapError(err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var config ReqConfig
	err = dec.Decode(&config)

	return &config, WrapError(err)
}

type ConfigChecker func(c ReqConfig) error

func (c ReqConfig) Check(checks ...ConfigChecker) error {
	for _, check := range checks {
		if err := check(c); err != nil {
			return WrapError(err)
		}
	}
	return nil
}

func URLCheck(c ReqConfig) error {
	if c.Url == "" {
		return WrapError(errors.New("url empty"))
	}
	_, err := url.Parse(c.Url)
	if err != nil {
		return WrapError(err)
	}
	return nil
}

func doServiceCheck(service XroadService) error {
	if service.XRoadInstance == "" {
		return WrapError(errors.New("Service xRoadInstance empty"))
	}
	if service.MemberClass == "" {
		return WrapError(errors.New("Service MemberClass empty"))
	}
	if service.MemberCode == "" {
		return WrapError(errors.New("Service MemberCode empty"))
	}
	if service.SubsystemCode == "" {
		return WrapError(errors.New("Service SubsystemCode empty"))
	}
	return nil
}

func ServiceCheck(c ReqConfig) error {
	// I assume ServiceCode and ServiceVersion is often empty in config to be set in runtime
	if service := c.SOAPHeader.Service; service != nil {
		return WrapError(doServiceCheck(*service))
	}
	return WrapError(errors.New("Service empty"))
}

func doCentralServiceCheck(centralService XroadCentralService) error {
	if centralService.XRoadInstance == "" {
		return WrapError(errors.New("CentralService xRoadInstance empty"))
	}
	if centralService.ServiceCode == "" {
		return WrapError(errors.New("CentralService ServiceCode empty"))
	}
	return nil
}

func CentralServiceCheck(c ReqConfig) error {
	if centralService := c.SOAPHeader.CentralService; centralService != nil {
		return WrapError(doCentralServiceCheck(*centralService))
	}
	return WrapError(errors.New("CentralService empty"))
}

func ServiceOrCentralServiceCheck(c ReqConfig) error {
	service := c.SOAPHeader.Service
	if service != nil {
		if err := doServiceCheck(*service); err != nil {
			return WrapError(err)
		}
	}
	centralService := c.SOAPHeader.CentralService
	if centralService != nil {
		if err := doCentralServiceCheck(*centralService); err != nil {
			return WrapError(err)
		}
	}
	if service == nil && centralService == nil {
		return WrapError(errors.New("Service and CentralService empty"))
	}
	if service != nil && centralService != nil {
		Log.Error("msg", "both service and centralService defined, centralService is used")
	}
	return nil
}

func ClientCheck(c ReqConfig) error {
	if c.SOAPHeader.Client.XRoadInstance == "" {
		return WrapError(errors.New("Client xRoadInstance empty"))
	}
	if c.SOAPHeader.Client.MemberClass == "" {
		return WrapError(errors.New("Client MemberClass empty"))
	}
	if c.SOAPHeader.Client.MemberCode == "" {
		return WrapError(errors.New("Client MemberCode empty"))
	}
	if c.SOAPHeader.Client.SubsystemCode == "" {
		return WrapError(errors.New("Client SubsystemCode empty"))
	}
	return nil
}
