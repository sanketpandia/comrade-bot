package membership

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrMemberNotFound     = errors.New("member not found")
	ErrInvalidRole        = errors.New("invalid role")
	ErrRoleChangeDenied   = errors.New("role change denied")
	ErrRemovalDenied      = errors.New("removal denied")
	ErrCannotRemoveAdmin  = errors.New("cannot remove administrator")
	ErrCannotRemoveSelf   = errors.New("cannot remove yourself")
)

func (s *Service) ListForVA(ctx context.Context, vaID string, active bool) ([]Member, error) {
	return s.memberships.ListForVA(ctx, vaID, active)
}

type UpdateRoleInput struct {
	ActorUserID   string
	TargetUserID  string
	VAID          string
	NewRole       string
}

func (s *Service) UpdateRole(ctx context.Context, input UpdateRoleInput) error {
	if !IsMemberRole(input.NewRole) {
		return ErrInvalidRole
	}
	target, err := s.memberships.GetForUserAndVA(ctx, input.TargetUserID, input.VAID)
	if err != nil {
		return err
	}
	if target == nil || !target.IsActive {
		return ErrMemberNotFound
	}

	if input.ActorUserID == input.TargetUserID {
		if input.NewRole != RoleAdministrator || target.Role != RoleAdministrator {
			return ErrRoleChangeDenied
		}
		return nil
	}

	demoteAdmin := ""
	if input.NewRole == RoleAdministrator {
		adminID, err := s.memberships.ActiveAdministratorUserID(ctx, input.VAID)
		if err != nil {
			return err
		}
		if adminID != "" && adminID != input.TargetUserID {
			demoteAdmin = adminID
		}
	}

	if target.Role == input.NewRole && demoteAdmin == "" {
		return nil
	}

	err = s.memberships.UpdateRoleForVA(ctx, input.VAID, input.TargetUserID, input.NewRole, demoteAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMemberNotFound
	}
	return err
}

type RemoveMemberInput struct {
	ActorUserID  string
	ActorRole    string
	TargetUserID string
	VAID         string
}

func (s *Service) RemoveMember(ctx context.Context, input RemoveMemberInput) error {
	if input.ActorUserID == input.TargetUserID {
		return ErrCannotRemoveSelf
	}
	target, err := s.memberships.GetForUserAndVA(ctx, input.TargetUserID, input.VAID)
	if err != nil {
		return err
	}
	if target == nil || !target.IsActive {
		return ErrMemberNotFound
	}
	if target.Role == RoleAdministrator {
		return ErrCannotRemoveAdmin
	}

	switch input.ActorRole {
	case RoleAdministrator:
		if target.Role != RoleProletariat && target.Role != RoleBourgeoisie {
			return ErrRemovalDenied
		}
	case RoleBourgeoisie:
		if target.Role != RoleProletariat {
			return ErrRemovalDenied
		}
	default:
		return ErrRemovalDenied
	}

	err = s.memberships.DeactivateForVA(ctx, input.VAID, input.TargetUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMemberNotFound
	}
	return err
}
