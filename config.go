package xroad

import (
	"encoding/json"
	"os"
)

type Config struct {
	SOAPHeader SOAPHeader `json:"header" mapstructure:"header"`
}

func LoadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, WrapError(err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var config Config
	err = dec.Decode(&config)

	return &config, WrapError(err)
}
