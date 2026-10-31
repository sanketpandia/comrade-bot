package signedlink

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/routes"
)

const (
	linkTypeDashboard = "DASHBOARD"
	linkTypeLiveMap     = "LIVE_MAP"
	linkTypeCustom      = "CUSTOM"
)

var (
	ErrMissingLinkType        = errors.New("missing link type")
	ErrInvalidLinkType        = errors.New("invalid link type")
	ErrMissingRedirect        = errors.New("missing redirect")
	ErrInvalidRedirect        = errors.New("invalid redirect")
	ErrUserNotFound           = errors.New("user not found")
	ErrNotVAMember            = errors.New("not va member")
	ErrUserLookupFailed       = errors.New("user lookup failed")
	ErrVALookupFailed         = errors.New("va lookup failed")
	ErrMembershipLookupFailed = errors.New("membership lookup failed")
	ErrMapLinkFailed          = errors.New("map link failed")
	ErrTicketIssueFailed      = errors.New("ticket issue failed")
)

type GenerateResult struct {
	URL        string
	RedirectTo string
	ExpiresIn  int
}

type vaMapLinkResolver interface {
	ActiveFlightsMapPath(ctx context.Context, vaID string) (string, error)
}

func (s *Service) Generate(ctx context.Context, claims auth.Claims, linkType, customRedirect string) (GenerateResult, error) {
	linkType = strings.TrimSpace(linkType)
	if err := validateLinkType(linkType); err != nil {
		return GenerateResult{}, err
	}

	user, err := s.users.GetByDiscordID(ctx, claims.DsUserID)
	if err != nil {
		slog.Error("lookup user for signed link", "error", err, "discord_user_id", claims.DsUserID, "link_type", linkType)
		return GenerateResult{}, fmt.Errorf("%w: %w", ErrUserLookupFailed, err)
	}
	if user == nil {
		return GenerateResult{}, ErrUserNotFound
	}

	vaID, role := "", ""
	if claims.DsServerID != "" && s.vaLookup != nil {
		configured, err := s.vaLookup.IsConfiguredVA(ctx, claims.DsServerID)
		if err != nil {
			slog.Error("lookup va for signed link", "error", err, "discord_user_id", claims.DsUserID, "link_type", linkType)
			return GenerateResult{}, fmt.Errorf("%w: %w", ErrVALookupFailed, err)
		}
		if configured && s.membership != nil {
			resolvedVA, resolvedRole, ok, err := s.membership.ResolveMembership(ctx, claims.DsUserID, claims.DsServerID)
			if err != nil {
				slog.Error("resolve membership for signed link", "error", err, "discord_user_id", claims.DsUserID, "link_type", linkType)
				return GenerateResult{}, fmt.Errorf("%w: %w", ErrMembershipLookupFailed, err)
			}
			if !ok {
				return GenerateResult{}, ErrNotVAMember
			}
			vaID, role = resolvedVA, resolvedRole
		}
	}

	redirectTo, err := s.resolveRedirect(ctx, linkType, vaID, customRedirect)
	if err != nil {
		return GenerateResult{}, err
	}

	token, err := s.tickets.Issue(ctx, auth.LoginTicket{
		UserID:          user.ID,
		DiscordUserID:   user.DiscordID,
		DiscordServerID: claims.DsServerID,
		VaID:            vaID,
		Role:            role,
		Username:        user.DisplayName(),
		RedirectTo:      redirectTo,
	})
	if err != nil {
		slog.Error("issue signed link ticket", "error", err, "discord_user_id", claims.DsUserID, "redirect_to", redirectTo, "link_type", linkType)
		return GenerateResult{}, fmt.Errorf("%w: %w", ErrTicketIssueFailed, err)
	}

	slog.Info("signed link issued",
		"link_type", linkType,
		"discord_user_id", claims.DsUserID,
		"user_id", user.ID,
		"redirect_to", redirectTo,
		"discord_server_id", claims.DsServerID,
		"va_id", vaID,
	)

	return GenerateResult{
		URL:        auth.FormatLoginURL(s.uiBaseURL, token),
		RedirectTo: redirectTo,
		ExpiresIn:  int(auth.TicketTTL.Seconds()),
	}, nil
}

func validateLinkType(linkType string) error {
	if linkType == "" {
		return ErrMissingLinkType
	}
	switch linkType {
	case linkTypeDashboard, linkTypeLiveMap, linkTypeCustom:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLinkType, linkType)
	}
}

func (s *Service) resolveRedirect(ctx context.Context, linkType, vaID, customRedirect string) (string, error) {
	switch linkType {
	case linkTypeDashboard:
		return routes.Dashboard, nil
	case linkTypeLiveMap:
		if vaID == "" {
			return routes.MapsFlightsActive, nil
		}
		if s.mapLink == nil {
			return routes.MapsFlightsActive, nil
		}
		path, err := s.mapLink.ActiveFlightsMapPath(ctx, vaID)
		if err != nil {
			slog.Error("build live map redirect for signed link", "error", err, "va_id", vaID)
			return "", fmt.Errorf("%w: %w", ErrMapLinkFailed, err)
		}
		normalized, err := auth.NormalizeRedirect(path)
		if err != nil {
			return "", ErrInvalidRedirect
		}
		return normalized, nil
	case linkTypeCustom:
		customRedirect = strings.TrimSpace(customRedirect)
		if customRedirect == "" {
			return "", ErrMissingRedirect
		}
		path, err := auth.NormalizeRedirect(customRedirect)
		if err != nil {
			return "", ErrInvalidRedirect
		}
		return path, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidLinkType, linkType)
	}
}

type Service struct {
	users      userLookup
	tickets    ticketIssuer
	membership membershipResolver
	vaLookup   vaByDiscordServer
	mapLink    vaMapLinkResolver
	uiBaseURL  string
}

func NewService(users userLookup, tickets ticketIssuer, membership membershipResolver, vaLookup vaByDiscordServer, mapLink vaMapLinkResolver, uiBaseURL string) *Service {
	return &Service{users: users, tickets: tickets, membership: membership, vaLookup: vaLookup, mapLink: mapLink, uiBaseURL: uiBaseURL}
}
