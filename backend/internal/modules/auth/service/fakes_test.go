package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/auth/repository"
	"hms-backend/internal/pkg/pagination"
)

// The fakes below are minimal, in-memory implementations of the auth
// module's repository/notifier interfaces used to unit test authService's
// business logic without a real MongoDB instance.

type fakeUserRepository struct {
	mu    sync.Mutex
	byID  map[string]*domain.User
	nextN int
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byID: make(map[string]*domain.User)}
}

func (f *fakeUserRepository) Create(_ context.Context, user *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextN++
	user.ID = idFromInt(f.nextN)
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	clone := *user
	f.byID[user.ID.Hex()] = &clone
	return nil
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range f.byID {
		if u.Email == email {
			clone := *u
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *u
	return &clone, nil
}

func (f *fakeUserRepository) IsActiveUser(ctx context.Context, id string) (bool, error) {
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return false, nil
	}
	return u.IsActive(), nil
}

func (f *fakeUserRepository) UpdatePasswordHash(_ context.Context, id primitive.ObjectID, passwordHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

func (f *fakeUserRepository) UpdateLastLoginAt(_ context.Context, id primitive.ObjectID, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	u.LastLoginAt = &at
	return nil
}

func (f *fakeUserRepository) List(_ context.Context, roleCode string, page pagination.Request) ([]domain.User, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []domain.User
	for _, u := range f.byID {
		if roleCode != "" && u.RoleCode != roleCode {
			continue
		}
		matched = append(matched, *u)
	}

	total := int64(len(matched))
	start := int(page.Skip())
	if start > len(matched) {
		start = len(matched)
	}
	end := start + int(page.Limit())
	if end > len(matched) {
		end = len(matched)
	}

	return matched[start:end], total, nil
}

// seed inserts a user directly (bypassing Create) for test setup.
func (f *fakeUserRepository) seed(user *domain.User) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextN++
	user.ID = idFromInt(f.nextN)
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	clone := *user
	f.byID[user.ID.Hex()] = &clone
}

type fakeRoleRepository struct {
	byCode map[string]*domain.Role
}

func newFakeRoleRepository() *fakeRoleRepository {
	roles := make(map[string]*domain.Role)
	for code, perms := range domain.DefaultRolePermissions {
		roles[code] = &domain.Role{Code: code, Name: domain.DisplayName(code), Permissions: perms}
	}
	return &fakeRoleRepository{byCode: roles}
}

func (f *fakeRoleRepository) FindByCode(_ context.Context, code string) (*domain.Role, error) {
	r, ok := f.byCode[code]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *r
	return &clone, nil
}

func (f *fakeRoleRepository) List(_ context.Context) ([]domain.Role, error) {
	var out []domain.Role
	for _, r := range f.byCode {
		out = append(out, *r)
	}
	return out, nil
}

func (f *fakeRoleRepository) Upsert(_ context.Context, role *domain.Role) error {
	clone := *role
	f.byCode[role.Code] = &clone
	return nil
}

type fakeRefreshTokenRepository struct {
	mu    sync.Mutex
	byID  map[string]*domain.RefreshToken
	nextN int
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{byID: make(map[string]*domain.RefreshToken)}
}

func (f *fakeRefreshTokenRepository) Create(_ context.Context, token *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextN++
	token.ID = idFromInt(f.nextN)
	token.CreatedAt = time.Now().UTC()
	clone := *token
	f.byID[token.ID.Hex()] = &clone
	return nil
}

func (f *fakeRefreshTokenRepository) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, t := range f.byID {
		if t.TokenHash == hash {
			clone := *t
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeRefreshTokenRepository) Revoke(_ context.Context, id primitive.ObjectID, replacedByHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	t.RevokedAt = &now
	t.ReplacedByHash = replacedByHash
	return nil
}

func (f *fakeRefreshTokenRepository) RevokeAllForUser(_ context.Context, userID primitive.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC()
	for _, t := range f.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRefreshTokenRepository) countActiveForUser(userID primitive.ObjectID) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	n := 0
	for _, t := range f.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			n++
		}
	}
	return n
}

type fakePasswordResetRepository struct {
	mu    sync.Mutex
	byID  map[string]*domain.PasswordResetToken
	nextN int
}

func newFakePasswordResetRepository() *fakePasswordResetRepository {
	return &fakePasswordResetRepository{byID: make(map[string]*domain.PasswordResetToken)}
}

func (f *fakePasswordResetRepository) Create(_ context.Context, token *domain.PasswordResetToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextN++
	token.ID = idFromInt(f.nextN)
	token.CreatedAt = time.Now().UTC()
	clone := *token
	f.byID[token.ID.Hex()] = &clone
	return nil
}

func (f *fakePasswordResetRepository) FindByHash(_ context.Context, hash string) (*domain.PasswordResetToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, t := range f.byID {
		if t.TokenHash == hash {
			clone := *t
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

// expireAll forces every stored token to be already expired, for testing
// the "expired token" rejection path without waiting on the real TTL.
func (f *fakePasswordResetRepository) expireAll() {
	f.mu.Lock()
	defer f.mu.Unlock()

	past := time.Now().UTC().Add(-time.Hour)
	for _, t := range f.byID {
		t.ExpiresAt = past
	}
}

func (f *fakePasswordResetRepository) MarkUsed(_ context.Context, id primitive.ObjectID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, ok := f.byID[id.Hex()]
	if !ok {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	return nil
}

type fakeNotifier struct {
	mu          sync.Mutex
	resetEmails []string
	resetLinks  []string
}

func (f *fakeNotifier) SendPasswordResetEmail(_ context.Context, toEmail, _, resetLink string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resetEmails = append(f.resetEmails, toEmail)
	f.resetLinks = append(f.resetLinks, resetLink)
	return nil
}

func (f *fakeNotifier) lastResetLink() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.resetLinks) == 0 {
		return ""
	}
	return f.resetLinks[len(f.resetLinks)-1]
}

// idFromInt returns a fresh, unique ObjectID. The mongo-driver's generator
// combines a per-process counter with the timestamp, so distinct calls are
// guaranteed unique regardless of n; n only seeds the timestamp.
func idFromInt(n int) primitive.ObjectID {
	return primitive.NewObjectIDFromTimestamp(time.Unix(int64(1_700_000_000+n), 0))
}
