package xroad

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
)

type ReqConfig struct {
	Url        string     `json:"url" mapstructure:"url"`
	SOAPHeader SOAPHeader `json:"header" mapstructure:"header"`
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

func ServiceCheck(c ReqConfig) error {
	// I assume ServiceCode and ServiceVersion is often empty in config to be set in runtime
	if c.SOAPHeader.Service.XRoadInstance == "" {
		return WrapError(errors.New("xRoadInstance empty"))
	}
	if c.SOAPHeader.Service.MemberClass == "" {
		return WrapError(errors.New("MemberClass empty"))
	}
	if c.SOAPHeader.Service.MemberCode == "" {
		return WrapError(errors.New("MemberCode empty"))
	}
	if c.SOAPHeader.Service.SubsystemCode == "" {
		return WrapError(errors.New("SubsystemCode empty"))
	}
	return nil
}

func ClientCheck(c ReqConfig) error {
	if c.SOAPHeader.Client.XRoadInstance == "" {
		return WrapError(errors.New("xRoadInstance empty"))
	}
	if c.SOAPHeader.Client.MemberClass == "" {
		return WrapError(errors.New("MemberClass empty"))
	}
	if c.SOAPHeader.Client.MemberCode == "" {
		return WrapError(errors.New("MemberCode empty"))
	}
	if c.SOAPHeader.Client.SubsystemCode == "" {
		return WrapError(errors.New("SubsystemCode empty"))
	}
	return nil
}
