package userstatus

import (
	"context"

	"infinite-experiment/politburo/internal/membership"
	"infinite-experiment/politburo/internal/users"
	"infinite-experiment/politburo/internal/virtualairlines"
)

type Builder struct {
	users       *users.Repository
	memberships *membership.Repository
	vas         *virtualairlines.Repository
}

func NewBuilder(users *users.Repository, memberships *membership.Repository, vas *virtualairlines.Repository) *Builder {
	return &Builder{users: users, memberships: memberships, vas: vas}
}

type Status struct {
	IsRegistered         bool
	GlobalUserExists     bool
	UserID               string
	DiscordID            string
	IFCommunityID        string
	IFAPIID              string
	IsActive             bool
	CreatedAt            string
	Affiliations         []Affiliation
	CurrentServer        CurrentServer
	CurrentVA            CurrentVA
	MembershipsSummary   Summary
	OtherMemberships     []OtherMembership
	OtherMembershipsCount int
}

type Affiliation struct {
	VAID     string
	VAName   string
	VACode   string
	Role     string
	IsActive bool
	JoinedAt string
	Callsign string
}

type CurrentServer struct {
	DiscordServerID string
	IsConfiguredVA  bool
	VAID            string
	VAName          string
	VACode          string
}

type CurrentVA struct {
	IsMember bool
	VAID     string
	VAName   string
	VACode   string
	Role     string
	IsActive bool
	Callsign string
}

type Summary struct {
	TotalCount  int
	ActiveCount int
}

type OtherMembership struct {
	VAID     string
	VAName   string
	VACode   string
	Role     string
	IsActive bool
}

func (b *Builder) Build(ctx context.Context, discordID, discordServerID string) (*Status, error) {
	user, err := b.users.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, err
	}

	status := &Status{DiscordID: discordID}
	if user == nil {
		status.IsRegistered = false
		status.GlobalUserExists = false
	} else {
		status.IsRegistered = true
		status.GlobalUserExists = true
		status.UserID = user.ID
		status.IsActive = user.IsActive
		if user.IFCommunityID != nil {
			status.IFCommunityID = *user.IFCommunityID
		}
		if user.IFApiID != nil {
			status.IFAPIID = *user.IFApiID
		}
		status.CreatedAt = user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	if discordServerID != "" {
		status.CurrentServer.DiscordServerID = discordServerID
		va, err := b.vas.GetByDiscordServerID(ctx, discordServerID)
		if err != nil {
			return nil, err
		}
		if va != nil {
			status.CurrentServer.IsConfiguredVA = true
			status.CurrentServer.VAID = va.ID
			status.CurrentServer.VAName = va.Name
			status.CurrentServer.VACode = va.Code
		}
	}

	if user != nil {
		memberships, err := b.memberships.ListForUser(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			aff := Affiliation{
				VAID: m.VAID, VAName: m.VAName, VACode: m.VACode,
				Role: m.Role, IsActive: m.IsActive, JoinedAt: m.JoinedAt.UTC().Format("2006-01-02T15:04:05Z"),
			}
			if m.Callsign != nil {
				aff.Callsign = *m.Callsign
			}
			status.Affiliations = append(status.Affiliations, aff)
			status.MembershipsSummary.TotalCount++
			if m.IsActive {
				status.MembershipsSummary.ActiveCount++
			}
			if status.CurrentServer.VAID != "" && m.VAID == status.CurrentServer.VAID {
				status.CurrentVA.IsMember = true
				status.CurrentVA.VAID = m.VAID
				status.CurrentVA.VAName = m.VAName
				status.CurrentVA.VACode = m.VACode
				status.CurrentVA.Role = m.Role
				status.CurrentVA.IsActive = m.IsActive
				if m.Callsign != nil {
					status.CurrentVA.Callsign = *m.Callsign
				}
			} else if m.IsActive {
				status.OtherMemberships = append(status.OtherMemberships, OtherMembership{
					VAID: m.VAID, VAName: m.VAName, VACode: m.VACode, Role: m.Role, IsActive: m.IsActive,
				})
			}
		}
		status.OtherMembershipsCount = len(status.OtherMemberships)
	}

	return status, nil
}
