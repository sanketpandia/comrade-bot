package users

import "time"

// User is a row from public.users.
type User struct {
	ID            string
	DiscordID     string
	IFCommunityID *string
	IFApiID       *string
	IsActive      bool
	Username      *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (u User) DisplayName() string {
	if u.Username != nil && *u.Username != "" {
		return *u.Username
	}
	if u.IFCommunityID != nil {
		return *u.IFCommunityID
	}
	return ""
}
