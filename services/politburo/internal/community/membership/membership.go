package membership

import "time"

type Membership struct {
	ID               string
	UserID           string
	VAID             string
	Role             string
	IsActive         bool
	Callsign         *string
	AirtablePilotID  *string
	JoinedAt         time.Time
	VAName           string
	VACode           string
}
