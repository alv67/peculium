package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// fakeSettingsRepo is an in-memory stand-in for repository.SettingsRepository
// serving the singleton row and recording the registration lock.
type fakeSettingsRepo struct {
	settings *model.ServerSettings
	lockHits int
	lastSet  *model.ServerSettings
}

func (f *fakeSettingsRepo) current() *model.ServerSettings {
	if f.settings == nil {
		return &model.ServerSettings{AutoApproveRegistrations: true}
	}
	s := *f.settings
	return &s
}

func (f *fakeSettingsRepo) Get(ctx context.Context) (*model.ServerSettings, error) {
	return f.current(), nil
}
func (f *fakeSettingsRepo) GetForUpdate(ctx context.Context) (*model.ServerSettings, error) {
	f.lockHits++
	return f.current(), nil
}
func (f *fakeSettingsRepo) UpdateAutoApprove(ctx context.Context, autoApprove bool) (*model.ServerSettings, error) {
	s := &model.ServerSettings{AutoApproveRegistrations: autoApprove, UpdatedAt: time.Now()}
	f.settings = s
	f.lastSet = s
	return f.current(), nil
}

func newAdminTestService(u *fakeUserRepo, st *fakeSettingsRepo) *Service {
	repos := &repository.Repository{User: u, Settings: st}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
}

func TestRegister_FirstUserBecomesActiveAdmin(t *testing.T) {
	u := &fakeUserRepo{count: 0}
	st := &fakeSettingsRepo{settings: &model.ServerSettings{AutoApproveRegistrations: false}}
	svc := newAdminTestService(u, st)

	user, err := svc.Register(context.Background(), "first@example.com", "First", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Role != model.RoleAdmin {
		t.Fatalf("role = %q, want admin for the first user", user.Role)
	}
	if user.Status != model.StatusActive {
		t.Fatalf("status = %q, want active even with auto-approve off", user.Status)
	}
	if st.lockHits != 1 {
		t.Fatalf("settings lock hits = %d, want 1 (registration must serialize on the settings row)", st.lockHits)
	}
}

func TestRegister_SubsequentUserActiveWhenAutoApprove(t *testing.T) {
	u := &fakeUserRepo{count: 1}
	st := &fakeSettingsRepo{settings: &model.ServerSettings{AutoApproveRegistrations: true}}
	svc := newAdminTestService(u, st)

	user, err := svc.Register(context.Background(), "second@example.com", "Second", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Role != model.RoleViewer {
		t.Fatalf("role = %q, want viewer", user.Role)
	}
	if user.Status != model.StatusActive {
		t.Fatalf("status = %q, want active", user.Status)
	}
}

func TestRegister_SubsequentUserPendingWhenAutoApproveOff(t *testing.T) {
	u := &fakeUserRepo{count: 1}
	st := &fakeSettingsRepo{settings: &model.ServerSettings{AutoApproveRegistrations: false}}
	svc := newAdminTestService(u, st)

	user, err := svc.Register(context.Background(), "pending@example.com", "Pending", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Status != model.StatusPending {
		t.Fatalf("status = %q, want pending", user.Status)
	}
	if user.Role != model.RoleViewer {
		t.Fatalf("role = %q, want viewer", user.Role)
	}
}

func TestRegister_DuplicateEmailRejected(t *testing.T) {
	u := &fakeUserRepo{
		count: 1,
		user:  &model.User{ID: uuid.New(), Email: "taken@example.com", Status: model.StatusActive},
	}
	st := &fakeSettingsRepo{}
	svc := newAdminTestService(u, st)

	_, err := svc.Register(context.Background(), "taken@example.com", "Dup", "password123")
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("err = %v, want ErrEmailExists", err)
	}
}

func loginTestUser(status model.Status) *fakeUserRepo {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	return &fakeUserRepo{user: &model.User{
		ID: uuid.New(), Email: "u@example.com", PasswordHash: string(hash),
		Role: model.RoleViewer, Status: status,
	}}
}

func TestLogin_PendingAccountRejected(t *testing.T) {
	svc := newAdminTestService(loginTestUser(model.StatusPending), &fakeSettingsRepo{})

	_, _, _, err := svc.Login(context.Background(), "u@example.com", "password123")
	if !errors.Is(err, ErrAccountPending) {
		t.Fatalf("err = %v, want ErrAccountPending", err)
	}
}

func TestLogin_DisabledAccountRejected(t *testing.T) {
	svc := newAdminTestService(loginTestUser(model.StatusDisabled), &fakeSettingsRepo{})

	_, _, _, err := svc.Login(context.Background(), "u@example.com", "password123")
	if !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestLogin_BadPasswordStillInvalidCredentials(t *testing.T) {
	svc := newAdminTestService(loginTestUser(model.StatusPending), &fakeSettingsRepo{})

	_, _, _, err := svc.Login(context.Background(), "u@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials (status must not leak before the password check)", err)
	}
}

func TestUpdateUser_InvalidRoleRejected(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New(), Role: model.RoleViewer, Status: model.StatusActive}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	role := model.Role("wizard")
	_, err := svc.UpdateUser(context.Background(), u.user.ID, &role, nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if len(u.roleWrites) != 0 {
		t.Fatal("invalid role must not reach the repository")
	}
}

func TestUpdateUser_InvalidStatusRejected(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New(), Role: model.RoleViewer, Status: model.StatusActive}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	status := model.Status("gone")
	_, err := svc.UpdateUser(context.Background(), u.user.ID, nil, &status)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestUpdateUser_NoFieldsRejected(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New(), Role: model.RoleEditor, Status: model.StatusActive}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	_, err := svc.UpdateUser(context.Background(), u.user.ID, nil, nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestUpdateUser_ApprovePendingEditor(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New(), Role: model.RoleEditor, Status: model.StatusPending}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	status := model.StatusActive
	user, err := svc.UpdateUser(context.Background(), u.user.ID, nil, &status)
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if user.Status != model.StatusActive {
		t.Fatalf("status = %q, want active after approval", user.Status)
	}
}

func TestUpdateUser_LastActiveAdminCannotBeDemoted(t *testing.T) {
	u := &fakeUserRepo{
		user:         &model.User{ID: uuid.New(), Email: "a@b.co", Role: model.RoleAdmin, Status: model.StatusActive},
		activeAdmins: 0,
	}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	role := model.RoleEditor
	_, err := svc.UpdateUser(context.Background(), u.user.ID, &role, nil)
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("err = %v, want ErrLastAdmin", err)
	}
	if u.user.Role != model.RoleAdmin {
		t.Fatalf("role = %q, want the untouched admin", u.user.Role)
	}
}

func TestUpdateUser_OwnerCountsAsAdminForTheGuard(t *testing.T) {
	u := &fakeUserRepo{
		user:         &model.User{ID: uuid.New(), Role: model.RoleOwner, Status: model.StatusActive},
		activeAdmins: 0,
	}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	status := model.StatusDisabled
	_, err := svc.UpdateUser(context.Background(), u.user.ID, nil, &status)
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("err = %v, want ErrLastAdmin (owner is admin-equivalent)", err)
	}
}

func TestUpdateUser_DemoteAllowedWhenAnotherActiveAdminExists(t *testing.T) {
	u := &fakeUserRepo{
		user:         &model.User{ID: uuid.New(), Role: model.RoleAdmin, Status: model.StatusActive},
		activeAdmins: 1,
	}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	role := model.RoleViewer
	status := model.StatusDisabled
	user, err := svc.UpdateUser(context.Background(), u.user.ID, &role, &status)
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if user.Role != model.RoleViewer || user.Status != model.StatusDisabled {
		t.Fatalf("got role=%q status=%q, want viewer/disabled", user.Role, user.Status)
	}
	if len(u.roleWrites) != 1 || len(u.statusWrites) != 1 {
		t.Fatalf("writes = %v/%v, want one role and one status write", u.roleWrites, u.statusWrites)
	}
}

func TestUpdateUser_UnknownUserNotFound(t *testing.T) {
	u := &fakeUserRepo{}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	_, err := svc.UpdateUser(context.Background(), uuid.New(), nil, ptrStatus(model.StatusActive))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAdminResetPassword(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New(), Email: "u@example.com", Status: model.StatusActive}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	if err := svc.AdminResetPassword(context.Background(), u.user.ID, "brand-new-password"); err != nil {
		t.Fatalf("AdminResetPassword: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte("brand-new-password")); err != nil {
		t.Fatalf("stored hash does not match the new password: %v", err)
	}
}

func TestAdminResetPassword_WeakPasswordRejected(t *testing.T) {
	u := &fakeUserRepo{user: &model.User{ID: uuid.New()}}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	if err := svc.AdminResetPassword(context.Background(), u.user.ID, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("err = %v, want ErrWeakPassword", err)
	}
	if u.passwordHash != "" {
		t.Fatal("a rejected reset must not write a hash")
	}
}

func TestAdminResetPassword_UnknownUserNotFound(t *testing.T) {
	u := &fakeUserRepo{}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	if err := svc.AdminResetPassword(context.Background(), uuid.New(), "long-enough-pass"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListUsers(t *testing.T) {
	rows := []*model.User{
		{ID: uuid.New(), Email: "a@b.co", Role: model.RoleAdmin, Status: model.StatusActive},
		{ID: uuid.New(), Email: "c@d.co", Role: model.RoleViewer, Status: model.StatusPending},
	}
	u := &fakeUserRepo{users: rows}
	svc := newAdminTestService(u, &fakeSettingsRepo{})

	users, err := svc.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 || users[1].Status != model.StatusPending {
		t.Fatalf("got %v, want the two stored rows", users)
	}
}

func TestGetAndUpdateSettings(t *testing.T) {
	st := &fakeSettingsRepo{settings: &model.ServerSettings{AutoApproveRegistrations: true}}
	svc := newAdminTestService(&fakeUserRepo{}, st)

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !settings.AutoApproveRegistrations {
		t.Fatal("auto approve = false, want the stored true")
	}

	updated, err := svc.UpdateSettings(context.Background(), false)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.AutoApproveRegistrations {
		t.Fatal("auto approve = true, want false after the update")
	}
}

func ptrStatus(s model.Status) *model.Status { return &s }
