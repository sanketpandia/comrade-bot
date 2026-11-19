package config

import (
	"encoding/json"
	"time"
)

const (
	TypeBasic           = "BASIC"
	TypeWebhooks        = "WEBHOOKS"
	TypeDataConnections = "DATA_CONNECTIONS"
	TypeMapStats        = "MAP_STATS"
	TypeMapPIREP        = "MAP_PIREP"
)

// PublicConfigTypes are allowed on the generic /va/config API.
var PublicConfigTypes = map[string]struct{}{
	TypeBasic:    {},
	TypeWebhooks: {},
}

type Block struct {
	VAID       string
	ConfigType string
	Config     json.RawMessage
	UpdatedAt  time.Time
	UpdatedBy  string
}
