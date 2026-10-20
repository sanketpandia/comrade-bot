package config

import (
	"encoding/json"
	"time"
)

const TypeBasic = "BASIC"

type Block struct {
	VAID       string
	ConfigType string
	Config     json.RawMessage
	UpdatedAt  time.Time
	UpdatedBy  string
}
