package session

import (
	"time"

	"infinite-experiment/politburo/internal/access/auth"
)

type Session struct {
	SessionID       string    `json:"session_id"`
	UserID          string    `json:"user_id"`
	DiscordID       string    `json:"discord_id"`
	DiscordServerID string    `json:"discord_server_id,omitempty"`
	VaID            string    `json:"va_id,omitempty"`
	Role            string    `json:"role,omitempty"`
	Username        string    `json:"username,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (s Session) Claims() auth.Claims {
	return auth.Claims{
		PbUserID:     s.UserID,
		DsUserID:     s.DiscordID,
		DsServerID:   s.DiscordServerID,
		PbServerID:   s.VaID,
		Role:         s.Role,
	}
}

func (s Session) Expired(now time.Time) bool {
	return now.After(s.ExpiresAt)
}
