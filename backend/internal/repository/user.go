package repository

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/alv67/peculium/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, email, name, password string, role model.Role, status model.Status) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context) ([]*model.User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role model.Role) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) error
	CountActiveAdmins(ctx context.Context, excluding uuid.UUID) (int64, error)
}

type userRepo struct {
	db DBTX
}

const userColumns = `id, email, name, password_hash, role, status, base_currency, created_at, updated_at`

func (r *userRepo) Create(ctx context.Context, email, name, password string, role model.Role, status model.Status) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{}
	err = r.db.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash, role, status) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, email, name, role, status, base_currency, created_at, updated_at`,
		email, name, string(hash), string(role), string(status),
	).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.Status, &user.BaseCurrency, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.Status, &user.BaseCurrency, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.Status, &user.BaseCurrency, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name = $1, email = $2, base_currency = $3, updated_at = NOW() WHERE id = $4`,
		user.Name, user.Email, user.BaseCurrency, user.ID,
	)
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, id,
	)
	return err
}

func (r *userRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (r *userRepo) List(ctx context.Context) ([]*model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.Status, &user.BaseCurrency, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *userRepo) UpdateRole(ctx context.Context, id uuid.UUID, role model.Role) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`,
		string(role), id,
	)
	return err
}

func (r *userRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`,
		string(status), id,
	)
	return err
}

func (r *userRepo) CountActiveAdmins(ctx context.Context, excluding uuid.UUID) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM users
		 WHERE role IN ('owner', 'admin') AND status = 'active' AND id <> $1`,
		excluding,
	).Scan(&n)
	return n, err
}
