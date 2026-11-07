package membership

import "time"

// Member is a VA membership row joined with user identity fields for staff listing.
type Member struct {
	UserID          string
	IFCommunityID   *string
	Username        *string
	Role            string
	Callsign        *string
	JoinedAt        time.Time
	UpdatedAt       time.Time
	IsActive        bool
	MembershipID    string
	AirtablePilotID *string
}
