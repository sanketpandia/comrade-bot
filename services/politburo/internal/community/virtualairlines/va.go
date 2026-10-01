package virtualairlines

import "time"

type VirtualAirline struct {
	ID              string
	Name            string
	Code            string
	DiscordServerID *string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
