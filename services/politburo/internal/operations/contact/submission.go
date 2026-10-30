package contact

import "time"

type Submission struct {
	ID              string
	IFCommunityID   string
	Message         string
	CreatedAt       time.Time
}
