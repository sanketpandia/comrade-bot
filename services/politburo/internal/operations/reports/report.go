package reports

import "time"

const (
	KindOccupiedIFC         = "occupied_ifc"
	KindMigrateDiscordServer = "migrate_discord_server"
	StatusOpen              = "open"
	StatusResolved          = "resolved"
)

type Report struct {
	ID                 string
	Kind               string
	Status             string
	ReporterDiscordID  string
	ClaimedIFC         *string
	Note               *string
	VAID               *string
	NewDiscordServerID *string
	CreatedAt          time.Time
	ResolvedAt         *time.Time
	ResolvedBy         *string
}
