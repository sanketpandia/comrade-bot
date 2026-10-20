package config

import "encoding/json"

func Validate(configType string, raw json.RawMessage) error {
	switch configType {
	case TypeBasic:
		return validateBasic(raw)
	default:
		return ErrUnknownType
	}
}
