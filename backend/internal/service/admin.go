package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

var userRoles = map[string]bool{
	string(model.RoleOwner): true, string(model.RoleAdmin): true,
	string(model.RoleEditor): true, string(model.RoleViewer): true,
}

var userStatuses = map[string]bool{
	string(model.StatusActive): true, string(model.StatusPending): true,
	string(model.StatusDisabled): true,
}

// ListUsers returns every account for the admin user-management screen.
func (s *Service) ListUsers(ctx context.Context) ([]*model.User, error) {
	return s.repos.User.List(ctx)
}

// UpdateUser applies the admin-supplied role and/or status to a user. At
// least one field must be provided and both must pass the enum validation.
// The change is guarded so the server never ends up without an active admin.
func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, role *model.Role, status *model.Status) (*model.User, error) {
	if role == nil && status == nil {
		return nil, fmt.Errorf("%w: role or status required", ErrInvalidInput)
	}
	if role != nil && !userRoles[string(*role)] {
		return nil, fmt.Errorf("%w: invalid role %q", ErrInvalidInput, *role)
	}
	if status != nil && !userStatuses[string(*status)] {
		return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidInput, *status)
	}

	var updated *model.User
	err := s.withTx(ctx, func(tx *repository.Repository) error {
		user, err := tx.User.FindByID(ctx, id)
		if err != nil {
			return ErrNotFound
		}

		newRole, newStatus := user.Role, user.Status
		if role != nil {
			newRole = *role
		}
		if status != nil {
			newStatus = *status
		}

		wasActiveAdmin := user.Role.IsAdmin() && user.Status == model.StatusActive
		staysActiveAdmin := newRole.IsAdmin() && newStatus == model.StatusActive
		if wasActiveAdmin && !staysActiveAdmin {
			others, err := tx.User.CountActiveAdmins(ctx, id)
			if err != nil {
				return err
			}
			if others == 0 {
				return ErrLastAdmin
			}
		}

		if role != nil && *role != user.Role {
			if err := tx.User.UpdateRole(ctx, id, *role); err != nil {
				return err
			}
		}
		if status != nil && *status != user.Status {
			if err := tx.User.UpdateStatus(ctx, id, *status); err != nil {
				return err
			}
		}

		fresh, err := tx.User.FindByID(ctx, id)
		if err != nil {
			return err
		}
		updated = fresh
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// AdminResetPassword sets a new password for the given user without requiring
// the current one.
func (s *Service) AdminResetPassword(ctx context.Context, id uuid.UUID, password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if _, err := s.repos.User.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repos.User.UpdatePassword(ctx, id, string(hash))
}

// GetSettings returns the singleton server settings.
func (s *Service) GetSettings(ctx context.Context) (*model.ServerSettings, error) {
	return s.repos.Settings.Get(ctx)
}

// UpdateSettings writes the registration auto-approval flag and returns the
// stored row after the update.
func (s *Service) UpdateSettings(ctx context.Context, autoApprove bool) (*model.ServerSettings, error) {
	return s.repos.Settings.UpdateAutoApprove(ctx, autoApprove)
}
