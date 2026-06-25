package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/service"
	gg "github.com/Pashteto/lia/protocols/gateguard"
)

// fakeGGClient fakes the GateGuard gRPC client.
type fakeGGClient struct {
	user      *gg.User
	err       error
	tokenResp *gg.TokenResponse
	signErr   error

	gotSignInUser *gg.User // captures the User passed to SignInOAuth
}

func (f *fakeGGClient) CheckAuth(_ context.Context, _ *gg.TokenRequest, _ ...grpc.CallOption) (*gg.User, error) {
	return f.user, f.err
}
func (f *fakeGGClient) SignInOAuth(_ context.Context, in *gg.User, _ ...grpc.CallOption) (*gg.TokenResponse, error) {
	f.gotSignInUser = in
	return f.tokenResp, f.signErr
}

// TestSigner_SignIn_SendsValidStatusAndRole guards against the demo-login 503:
// GateGuard maps an unset (Unknown=0) proto status to its internal "unsupported"
// sentinel and then panics stringifying it during user insert
// ("index out of range [2] with length 2"). Lia must send a valid, explicit
// Status (and Role) so the provisioned GateGuard user is well-formed.
func TestSigner_SignIn_SendsValidStatusAndRole(t *testing.T) {
	fake := &fakeGGClient{tokenResp: &gg.TokenResponse{Token: []byte("jwt-123")}}
	s := newSignerWithClient(fake)

	if _, err := s.SignIn(context.Background(), "demo@lia.test", "Demo"); err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}
	if fake.gotSignInUser == nil {
		t.Fatal("SignInOAuth was not called")
	}
	if fake.gotSignInUser.Status != gg.UserStatus_UserActive {
		t.Errorf("expected Status=UserActive (%d), got %v", gg.UserStatus_UserActive, fake.gotSignInUser.Status)
	}
	if fake.gotSignInUser.Role != gg.UserRole_UserRoleCommon {
		t.Errorf("expected Role=UserRoleCommon (%d), got %v", gg.UserRole_UserRoleCommon, fake.gotSignInUser.Role)
	}
	if fake.gotSignInUser.Email != "demo@lia.test" || fake.gotSignInUser.Name != "Demo" {
		t.Errorf("email/name mismatch: %+v", fake.gotSignInUser)
	}
}

func TestSigner_SignIn_ReturnsToken(t *testing.T) {
	s := newSignerWithClient(&fakeGGClient{tokenResp: &gg.TokenResponse{Token: []byte("jwt-123")}})

	tok, err := s.SignIn(context.Background(), "alice@example.com", "Alice")
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}
	if tok != "jwt-123" {
		t.Errorf("expected token jwt-123, got %q", tok)
	}
}

func TestSigner_SignIn_Error(t *testing.T) {
	s := newSignerWithClient(&fakeGGClient{signErr: fmt.Errorf("gateguard down")})

	if _, err := s.SignIn(context.Background(), "a@b.com", "A"); err == nil {
		t.Error("expected error when SignInOAuth fails")
	}
}

func TestSigner_SignIn_EmptyToken(t *testing.T) {
	s := newSignerWithClient(&fakeGGClient{tokenResp: &gg.TokenResponse{Token: nil}})

	if _, err := s.SignIn(context.Background(), "a@b.com", "A"); err == nil {
		t.Error("expected error when GateGuard returns an empty token")
	}
}

func TestGatekeeperValidator_MapsUserToClaims(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	v := newValidatorWithClient(&fakeGGClient{
		user: &gg.User{Uuid: id.Bytes(), Email: "alice@example.com", Name: "Alice"},
	})

	claims, err := v.Validate(context.Background(), "session-token")
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if claims.Email != "alice@example.com" || claims.Name != "Alice" {
		t.Errorf("claims mismatch: %+v", claims)
	}
	if claims.Subject != id.String() {
		t.Errorf("expected subject %s, got %s", id, claims.Subject)
	}
}

func TestGatekeeperValidator_CheckAuthError(t *testing.T) {
	v := newValidatorWithClient(&fakeGGClient{err: fmt.Errorf("invalid token")})

	claims, err := v.Validate(context.Background(), "bad-token")
	if err == nil {
		t.Error("expected error when CheckAuth fails")
	}
	if claims != nil {
		t.Errorf("expected nil claims, got %v", claims)
	}
}

// fakeValidator is a TokenValidator stub for testing CheckAuth without Gatekeeper.
type fakeValidator struct {
	claims *Claims
	err    error
}

func (f *fakeValidator) Validate(_ context.Context, _ string) (*Claims, error) {
	return f.claims, f.err
}

func TestAuth_CheckAuth_ProvisionsNewUser(t *testing.T) {
	created := false
	var createdEmail, createdName string
	svc := &mockService{
		GetUserByEmailFunc: func(_ context.Context, email string) (*models.User, error) {
			return nil, fmt.Errorf("%w: %s", service.ErrNotFound, email)
		},
		CreateUserFunc: func(_ context.Context, u *models.User) error {
			created = true
			createdEmail = u.Email
			createdName = u.Name
			return nil
		},
	}
	v := &fakeValidator{claims: &Claims{Subject: "sub-1", Email: "new@example.com", Name: "New Person"}}
	a := NewAuth(svc, false, nil, WithValidator(v))

	user, err := a.CheckAuth("Bearer good-token")
	if err != nil {
		t.Fatalf("CheckAuth returned error: %v", err)
	}
	if !created {
		t.Fatal("expected a new user to be provisioned")
	}
	if createdEmail != "new@example.com" || createdName != "New Person" {
		t.Errorf("provisioned user mismatch: email=%q name=%q", createdEmail, createdName)
	}
	if user.Email == nil || string(*user.Email) != "new@example.com" {
		t.Errorf("principal email mismatch: %v", user.Email)
	}
	if user.Name == nil || *user.Name != "New Person" {
		t.Errorf("principal name mismatch: %v", user.Name)
	}
	if user.UUID == "" {
		t.Error("principal UUID is empty")
	}
}

func TestAuth_CheckAuth_ReusesExistingUser(t *testing.T) {
	existing := uuid.Must(uuid.NewV4())
	svc := &mockService{
		GetUserByEmailFunc: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{UUID: existing, Email: "e@example.com", Name: "Existing", Status: models.UserActive}, nil
		},
		CreateUserFunc: func(_ context.Context, _ *models.User) error {
			t.Error("CreateUser should not be called for an existing user")
			return nil
		},
	}
	v := &fakeValidator{claims: &Claims{Subject: "sub-2", Email: "e@example.com", Name: "Existing"}}
	a := NewAuth(svc, false, nil, WithValidator(v))

	user, err := a.CheckAuth("Bearer good-token")
	if err != nil {
		t.Fatalf("CheckAuth returned error: %v", err)
	}
	if string(user.UUID) != existing.String() {
		t.Errorf("expected existing UUID %s, got %s", existing, user.UUID)
	}
}

func TestAuth_CheckAuth_ValidatorError(t *testing.T) {
	svc := &mockService{}
	v := &fakeValidator{err: fmt.Errorf("token expired")}
	a := NewAuth(svc, false, nil, WithValidator(v))

	user, err := a.CheckAuth("Bearer bad-token")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
	if user != nil {
		t.Errorf("expected nil user on validation failure, got %v", user)
	}
}

func TestAuth_CheckAuth_NoValidatorConfigured(t *testing.T) {
	svc := &mockService{}
	a := NewAuth(svc, false, nil) // non-mock, no validator wired

	user, err := a.CheckAuth("Bearer any-token")
	if err == nil {
		t.Error("expected error when no validator is configured, got nil")
	}
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
}

// mockService is a mock implementation of service.IService for testing.
type mockService struct {
	CreateUserFunc     func(ctx context.Context, user *models.User) error
	GetUserByEmailFunc func(ctx context.Context, email string) (*models.User, error)
}

func (m *mockService) CreateUser(_ context.Context, user *models.User) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(context.Background(), user)
	}
	return nil
}

func (m *mockService) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	if m.GetUserByEmailFunc != nil {
		return m.GetUserByEmailFunc(context.Background(), email)
	}
	return nil, nil
}

func TestNewAuth(t *testing.T) {
	tests := []struct {
		name        string
		adminEmails []string
		mocked      bool
	}{
		{
			name:        "create auth with no admins",
			mocked:      false,
			adminEmails: []string{},
		},
		{
			name:        "create auth with admins",
			mocked:      false,
			adminEmails: []string{"admin1@example.com", "admin2@example.com"},
		},
		{
			name:        "create auth in mock mode",
			mocked:      true,
			adminEmails: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{}
			auth := NewAuth(svc, tt.mocked, tt.adminEmails)

			if auth == nil {
				t.Fatal("NewAuth returned nil")
			}

			if auth.mocked != tt.mocked {
				t.Errorf("expected mocked=%v, got %v", tt.mocked, auth.mocked)
			}

			if len(auth.admins) != len(tt.adminEmails) {
				t.Errorf("expected %d admins, got %d", len(tt.adminEmails), len(auth.admins))
			}

			for _, email := range tt.adminEmails {
				if _, ok := auth.admins[email]; !ok {
					t.Errorf("admin email %s not found in admins map", email)
				}
			}
		})
	}
}

func TestAuth_CheckAuth_MockMode(t *testing.T) {
	svc := &mockService{}
	auth := NewAuth(svc, true, []string{})

	user, err := auth.CheckAuth("Bearer test-token")
	if err != nil {
		t.Fatalf("CheckAuth failed in mock mode: %v", err)
	}

	if user == nil {
		t.Fatal("CheckAuth returned nil user in mock mode")
	}

	if user.UUID == "" {
		t.Error("mock user has empty UUID")
	}

	if user.Email == nil || *user.Email == "" {
		t.Error("mock user has empty email")
	}

	if user.Name == nil || *user.Name == "" {
		t.Error("mock user has empty name")
	}

	if user.Status == nil || *user.Status == "" {
		t.Error("mock user has empty status")
	}
}

func TestAuth_CheckAuth_NonMockMode(t *testing.T) {
	svc := &mockService{}
	auth := NewAuth(svc, false, []string{})

	// Non-mock mode should return unauthorized (since gatekeeper is not integrated)
	user, err := auth.CheckAuth("Bearer valid-token")
	if err == nil {
		t.Error("expected error in non-mock mode, got nil")
	}

	if user != nil {
		t.Errorf("expected nil user on error, got %v", user)
	}
}

func TestAuth_CheckAuth_TokenPrefixStripping(t *testing.T) {
	svc := &mockService{}
	auth := NewAuth(svc, true, []string{})

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "token with Bearer prefix",
			token: "Bearer test-token",
		},
		{
			name:  "token without Bearer prefix",
			token: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := auth.CheckAuth(tt.token)
			if err != nil {
				t.Fatalf("CheckAuth failed: %v", err)
			}

			if user == nil {
				t.Fatal("CheckAuth returned nil user")
			}
		})
	}
}

func TestAuth_IsAdmin(t *testing.T) {
	svc := &mockService{}
	adminEmails := []string{"admin1@example.com", "admin2@example.com"}
	auth := NewAuth(svc, false, adminEmails)

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "admin email 1",
			email:    "admin1@example.com",
			expected: true,
		},
		{
			name:     "admin email 2",
			email:    "admin2@example.com",
			expected: true,
		},
		{
			name:     "non-admin email",
			email:    "user@example.com",
			expected: false,
		},
		{
			name:     "empty email",
			email:    "",
			expected: false,
		},
		{
			name:     "case sensitive check",
			email:    "ADMIN1@EXAMPLE.COM",
			expected: false, // emails are case-sensitive in our implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.IsAdmin(tt.email)
			if result != tt.expected {
				t.Errorf("IsAdmin(%s) = %v, expected %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestAuth_MockUser(t *testing.T) {
	svc := &mockService{}
	auth := NewAuth(svc, true, []string{})

	user := auth.mockUser()

	if user == nil {
		t.Fatal("mockUser returned nil")
	}

	// Check UUID format
	if user.UUID == "" {
		t.Error("mock user UUID is empty")
	}

	// Expected UUID (lowercase from gofrs/uuid)
	expectedUUID := "fa734dc4-22e6-41c5-a913-30c302c1ca68"
	if string(user.UUID) != expectedUUID {
		t.Errorf("expected UUID %s, got %s", expectedUUID, user.UUID)
	}

	// Check email
	if user.Email == nil {
		t.Fatal("mock user email is nil")
	}
	expectedEmail := "test@example.com"
	if string(*user.Email) != expectedEmail {
		t.Errorf("expected email %s, got %s", expectedEmail, *user.Email)
	}

	// Check name
	if user.Name == nil {
		t.Fatal("mock user name is nil")
	}
	if *user.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", *user.Name)
	}

	// Check status
	if user.Status == nil {
		t.Fatal("mock user status is nil")
	}
	if *user.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", *user.Status)
	}
}
