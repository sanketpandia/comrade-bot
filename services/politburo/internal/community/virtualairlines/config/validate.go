package config

import (
	"encoding/json"
	"fmt"

	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
)

func Validate(configType string, raw json.RawMessage) error {
	switch configType {
	case TypeBasic:
		return validateBasic(raw)
	case TypeWebhooks:
		if err := webhookconfig.Validate(raw); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
		return nil
	default:
		return ErrUnknownType
	}
}
