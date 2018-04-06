package xroad

import (
	"encoding/json"
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
