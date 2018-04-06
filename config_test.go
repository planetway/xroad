package xroad

import (
	"testing"
)

func TestConfig(t *testing.T) {
	_, err := LoadConfig("config.json.template")
	if err != nil {
		t.Errorf("%s", err)
	}
}
