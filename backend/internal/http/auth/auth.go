// Package auth provides authentication helpers for the HTTP module.
package auth

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	apierr "github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/http/models"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/service"
	"github.com/Pashteto/lia/pkg/logger"
)

// Claims is the identity a TokenValidator extracts from a validated token.
type Claims struct {
	Subject       string
	Email         string
	Name          string
	Role          string
	EmailVerified bool
	ExpiresAt     time.Time
}

// TokenValidator validates a bearer token and returns its claims. The Gatekeeper
// gRPC adapter implements this; tests use a fake. Keeping it an interface isolates
// the (still-to-be-confirmed) Gatekeeper contract from the auth orchestration.
type TokenValidator interface {
	Validate(ctx context.Context, token string) (*Claims, error)
}

// Auth handles token validation and maps an identity to a local user principal.
type Auth struct {
	service   service.IService
	validator TokenValidator
	admins    map[string]struct{}
	mocked    bool
}

// Option configures an Auth.
type Option func(*Auth)

// WithValidator wires the token validator used in non-mock mode.
func WithValidator(v TokenValidator) Option {
	return func(a *Auth) { a.validator = v }
}

// NewAuth creates a new Auth instance.
func NewAuth(svc service.IService, mocked bool, adminEmails []string, opts ...Option) *Auth {
	adminsMap := make(map[string]struct{})
	for _, email := range adminEmails {
		adminsMap[email] = struct{}{}
	}

	a := &Auth{
		service: svc,
		mocked:  mocked,
		admins:  adminsMap,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Authenticate validates the bearer token and returns the local domain user
// principal, with its role synced from the token's claims (GateGuard is the
// source of truth; users.role is a cache).
func (a *Auth) Authenticate(token string) (*domain.User, error) {
	if a.mocked {
		logger.Log().Info("using mock authentication (bypassing gatekeeper)")
		return a.mockDomainUser(), nil
	}

	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return nil, apierr.New(401, "unauthorized access or invalid credentials")
	}
	if a.validator == nil {
		logger.Log().Error("auth: no token validator configured (gatekeeper not wired)")
		return nil, apierr.New(401, "unauthorized access or invalid credentials")
	}

	claims, err := a.validator.Validate(context.Background(), token)
	if err != nil {
		logger.Log().Infof("auth: token rejected: %v", err)
		return nil, apierr.New(401, "unauthorized access or invalid credentials")
	}

	u, err := a.ensureUser(context.Background(), claims.Email, claims.Name, normalizeRole(claims.Role), claims.EmailVerified)
	if err != nil {
		logger.Log().Errorf("auth: user provisioning failed for subject %s: %v", claims.Subject, err)
		return nil, apierr.New(401, "unauthorized access or invalid credentials")
	}
	logger.Log().Infof("auth: authenticated subject %s", claims.Subject)
	return u, nil
}

// CheckAuth keeps the swagger-facing principal contract (api models.User).
func (a *Auth) CheckAuth(token string) (*models.User, error) {
	u, err := a.Authenticate(token)
	if err != nil {
		return nil, err
	}
	return toPrincipal(u), nil
}

// ensureUser looks up a local user by email, provisioning one just-in-time on
// first sight (keyed by the unique email column — no schema change needed).
// The role is synced from the authoritative claim if it has drifted.
func (a *Auth) ensureUser(ctx context.Context, email, name, role string, emailVerified bool) (*domain.User, error) {
	u, err := a.service.GetUserByEmail(ctx, email)
	if err != nil {
		if !stderrors.Is(err, service.ErrNotFound) {
			return nil, fmt.Errorf("lookup user: %w", err)
		}
		u = &domain.User{
			UUID:          uuid.Must(uuid.NewV4()),
			Email:         email,
			Name:          name,
			Status:        domain.UserActive,
			Role:          role,
			EmailVerified: emailVerified,
		}
		if err := a.service.CreateUser(ctx, u); err != nil {
			return nil, fmt.Errorf("provision user: %w", err)
		}
		return u, nil
	}
	// Existing user: sync the role from the authoritative claim if it drifted.
	if u.Role != role {
		if err := a.service.UpdateUserRole(ctx, u.UUID, role); err != nil {
			return nil, fmt.Errorf("sync user role: %w", err)
		}
		u.Role = role
	}
	u.EmailVerified = emailVerified
	return u, nil
}

// normalizeRole maps GateGuard's enum string names to Lia's bare role tokens.
// gg.UserRole.String() yields e.g. "UserRoleAdmin"; we store "admin".
func normalizeRole(r string) string {
	switch r {
	case "UserRoleAdmin", "admin":
		return "admin"
	default:
		return "common"
	}
}

// toPrincipal maps a domain user to the swagger principal model.
func toPrincipal(u *domain.User) *models.User {
	email := strfmt.Email(u.Email)
	name := u.Name
	status := u.Status.String()
	return &models.User{
		UUID:          strfmt.UUID(u.UUID.String()),
		Email:         &email,
		Name:          &name,
		Status:        &status,
		EmailVerified: u.EmailVerified,
	}
}

// IsAdmin checks if email belongs to admin user.
func (a *Auth) IsAdmin(email string) bool {
	_, ok := a.admins[email]
	return ok
}

// mockUser returns a mock swagger user for testing without gatekeeper.
func (a *Auth) mockUser() *models.User {
	return toPrincipal(a.mockDomainUser())
}

// mockDomainUser returns a mock domain user for testing without gatekeeper.
func (a *Auth) mockDomainUser() *domain.User {
	return &domain.User{
		UUID:          uuid.Must(uuid.FromString("FA734DC4-22E6-41C5-A913-30C302C1CA68")),
		Email:         "test@example.com",
		Name:          "Test User",
		Status:        domain.UserActive,
		Role:          "common",
		EmailVerified: true,
	}
}
