package repository

import (
	"context"

	"github.com/alv67/peculium/internal/model"
)

// SettingsRepository accesses the singleton server_settings row (id = 1).
type SettingsRepository interface {
	Get(ctx context.Context) (*model.ServerSettings, error)
	// GetForUpdate locks the row for the current transaction. Registration
	// takes this lock to serialize concurrent signups so the first-user
	// promotion and the auto-approve decision cannot interleave.
	GetForUpdate(ctx context.Context) (*model.ServerSettings, error)
	UpdateAutoApprove(ctx context.Context, autoApprove bool) (*model.ServerSettings, error)
}

type settingsRepo struct {
	db DBTX
}

func scanServerSettings(row interface{ Scan(dest ...any) error }) (*model.ServerSettings, error) {
	s := &model.ServerSettings{}
	err := row.Scan(&s.AutoApproveRegistrations, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *settingsRepo) Get(ctx context.Context) (*model.ServerSettings, error) {
	row := r.db.QueryRow(ctx,
		`SELECT auto_approve_registrations, updated_at FROM server_settings WHERE id = 1`)
	return scanServerSettings(row)
}

func (r *settingsRepo) GetForUpdate(ctx context.Context) (*model.ServerSettings, error) {
	row := r.db.QueryRow(ctx,
		`SELECT auto_approve_registrations, updated_at FROM server_settings WHERE id = 1 FOR UPDATE`)
	return scanServerSettings(row)
}

func (r *settingsRepo) UpdateAutoApprove(ctx context.Context, autoApprove bool) (*model.ServerSettings, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE server_settings SET auto_approve_registrations = $1, updated_at = NOW()
		 WHERE id = 1
		 RETURNING auto_approve_registrations, updated_at`,
		autoApprove,
	)
	return scanServerSettings(row)
}
