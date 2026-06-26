// Package http implements the HTTP transport module.
package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/gofrs/uuid"
	"github.com/justinas/alice"

	"github.com/Pashteto/lia/config"
	categoriesdomain "github.com/Pashteto/lia/internal/categories"
	eventsdomain "github.com/Pashteto/lia/internal/events"
	filesdomain "github.com/Pashteto/lia/internal/files"
	"github.com/Pashteto/lia/internal/grpcclient"
	"github.com/Pashteto/lia/internal/http/admin"
	"github.com/Pashteto/lia/internal/http/auth"
	"github.com/Pashteto/lia/internal/http/handlers"
	"github.com/Pashteto/lia/internal/http/middlewares"
	organizershttp "github.com/Pashteto/lia/internal/http/organizers"
	httpserver "github.com/Pashteto/lia/internal/http/server"
	"github.com/Pashteto/lia/internal/http/server/operations"
	"github.com/Pashteto/lia/internal/http/uploads"
	"github.com/Pashteto/lia/internal/moderation"
	organizersdomain "github.com/Pashteto/lia/internal/organizers"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
	"github.com/Pashteto/lia/internal/service"
	settingsdomain "github.com/Pashteto/lia/internal/settings"
	"github.com/Pashteto/lia/internal/storage"
	venuesdomain "github.com/Pashteto/lia/internal/venues"
	"github.com/Pashteto/lia/pkg/logger"
)

// Module implements module.Module interface for the HTTP server.
type Module struct {
	config     *config.HTTPConfig
	service    service.IService
	grpcClient grpcclient.IClient
	events     eventsdomain.Service
	categories categoriesdomain.Service
	venues     venuesdomain.Service
	files      filesdomain.Service
	storage    storage.Storage
	rsvp       rsvpdomain.Service
	moderation moderation.Service
	modReason  func(uuid.UUID) (string, error)
	organizers organizersdomain.Service
	settings   settingsdomain.Service
	server     *httpserver.Server
	api        *operations.LiaAPIAPI
	handler    *http.Handler
	auth       *auth.Auth
}

// NewModule creates a new HTTP module instance.
func NewModule(cfg *config.HTTPConfig, svc service.IService, grpcClient grpcclient.IClient) *Module {
	return &Module{
		config:     cfg,
		service:    svc,
		grpcClient: grpcClient,
	}
}

// SetEventsService injects the events domain service. Call before Init.
// When nil, the events endpoints are left unregistered (the generated API
// returns "not implemented" for them).
func (m *Module) SetEventsService(svc eventsdomain.Service) {
	m.events = svc
}

// SetCategoriesService injects the categories domain service. Call before Init.
func (m *Module) SetCategoriesService(svc categoriesdomain.Service) {
	m.categories = svc
}

// SetVenuesService injects the venues domain service. Call before Init.
func (m *Module) SetVenuesService(svc venuesdomain.Service) {
	m.venues = svc
}

// SetFilesService injects the files domain service. Call before Init.
// When nil, upload/serve endpoints return 404 (the swagger mux returns 404 for unregistered paths).
func (m *Module) SetFilesService(svc filesdomain.Service) {
	m.files = svc
}

// SetStorage injects the blob storage backend. Call before Init.
func (m *Module) SetStorage(store storage.Storage) {
	m.storage = store
}

// SetRsvpService injects the RSVP domain service. Call before Init.
// When nil, the RSVP endpoints are left unregistered (the generated API
// returns "not implemented" for them).
func (m *Module) SetRsvpService(svc rsvpdomain.Service) {
	m.rsvp = svc
}

// SetModeration injects the moderation service and the LatestReason lookup
// function (bound to the moderation repository). Call before Init.
func (m *Module) SetModeration(svc moderation.Service, reason func(uuid.UUID) (string, error)) {
	m.moderation = svc
	m.modReason = reason
}

// SetOrganizers injects the organizers domain service. Call before Init.
func (m *Module) SetOrganizers(svc organizersdomain.Service) { m.organizers = svc }

// SetSettings injects the app-settings service. Call before Init.
func (m *Module) SetSettings(svc settingsdomain.Service) { m.settings = svc }

// Name returns the module identifier.
func (m *Module) Name() string {
	return "http"
}

// Init initializes the HTTP module.
func (m *Module) Init(_ context.Context) error {
	logger.Log().Infof("initializing %s module", m.Name())

	// Initialize auth. In non-mock mode, wire the Gatekeeper token validator from
	// config; if it's absent, CheckAuth safely denies (no silent open access).
	var authOpts []auth.Option
	if !m.config.MockAuth && m.config.Gatekeeper != nil {
		timeout := 5 * time.Second
		if d, err := time.ParseDuration(m.config.Gatekeeper.Timeout); err == nil && d > 0 {
			timeout = d
		}
		validator, err := auth.NewGatekeeperValidator(auth.GatekeeperConfig{
			Address: m.config.Gatekeeper.Address,
			Timeout: timeout,
		})
		if err != nil {
			return fmt.Errorf("init gatekeeper validator: %w", err)
		}
		authOpts = append(authOpts, auth.WithValidator(validator))
	}
	m.auth = auth.NewAuth(m.service, m.config.MockAuth, m.config.AdminEmails, authOpts...)

	// Initialize API
	if err := m.initAPI(); err != nil {
		return fmt.Errorf("init API: %w", err)
	}

	// Initialize server
	if err := m.initServer(); err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	logger.Log().Infof("HTTP server configured on %s:%d", m.config.Host, m.config.Port)
	return nil
}

// Start begins module operation.
func (m *Module) Start(_ context.Context) error {
	logger.Log().Infof("starting %s module", m.Name())

	go func() {
		logger.Log().Infof("HTTP server listening on %s:%d", m.config.Host, m.config.Port)
		if err := m.server.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log().Errorf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(_ context.Context) error {
	logger.Log().Infof("stopping %s module", m.Name())

	if m.server != nil {
		if err := m.server.Shutdown(); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
	}

	logger.Log().Info("HTTP module stopped successfully")
	return nil
}

// HealthCheck returns module health status.
func (m *Module) HealthCheck(_ context.Context) error {
	// HTTP module is healthy if server is running
	// Could add a ping to actual server if needed
	return nil
}

// initAPI initializes the API and wires handlers.
func (m *Module) initAPI() error {
	// Load swagger spec
	swaggerSpec, err := loads.Analyzed(httpserver.SwaggerJSON, "")
	if err != nil {
		return fmt.Errorf("load swagger spec: %w", err)
	}

	// Create API instance
	api := operations.NewLiaAPIAPI(swaggerSpec)

	// Configure logger
	api.Logger = logger.Log().Infof

	// Configure auth
	api.JwtAuth = m.auth.CheckAuth

	// Register handlers - pass grpcClient for external service access
	api.UsersGetUserByEmailHandler = handlers.NewGetUserByEmail(m.service, m.grpcClient)
	api.HealthGetHealthHandler = handlers.NewHealth()

	// Demo-login (DEMO-ONLY): mints GateGuard tokens via SignInOAuth. The signer
	// is wired only when gatekeeper is configured; otherwise the handler 503s.
	var signer auth.Signer
	if m.config.Gatekeeper != nil {
		timeout := 5 * time.Second
		if d, err := time.ParseDuration(m.config.Gatekeeper.Timeout); err == nil && d > 0 {
			timeout = d
		}
		s, err := auth.NewSigner(auth.GatekeeperConfig{Address: m.config.Gatekeeper.Address, Timeout: timeout})
		if err != nil {
			return fmt.Errorf("init demo-login signer: %w", err)
		}
		signer = s
	}
	api.AuthDemoLoginHandler = handlers.NewDemoLogin(signer)
	api.AuthRegisterHandler = handlers.NewRegister(signer)
	api.AuthLoginHandler = handlers.NewLogin(signer)

	// Events domain handlers (registered only when the events service is wired,
	// i.e. when the database module is enabled).
	if m.events != nil {
		api.EventsListEventsHandler = handlers.NewListEvents(m.events)
		api.EventsListMyEventsHandler = handlers.NewListMyEvents(m.events)
		api.EventsGetEventByIDHandler = handlers.NewGetEventByID(m.events, m.auth.CheckAuth)
		api.EventsCreateEventHandler = handlers.NewCreateEvent(m.events)
		api.EventsNearbyEventsHandler = handlers.NewNearbyEvents(m.events)
		api.EventsUpdateEventHandler = handlers.NewUpdateEvent(m.events)
	}

	if m.rsvp != nil {
		api.RsvpSignUpHandler = handlers.NewSignUp(m.rsvp)
		api.RsvpCancelRsvpHandler = handlers.NewCancelRsvp(m.rsvp)
		api.RsvpMyPracticesHandler = handlers.NewMyPractices(m.rsvp)
		api.RsvpMyApplicationsHandler = handlers.NewMyApplications(m.rsvp)
		api.RsvpListEventApplicationsHandler = handlers.NewListEventApplications(m.rsvp)
		api.RsvpDecideApplicationHandler = handlers.NewDecideApplication(m.rsvp)
		api.RsvpEventCalendarHandler = handlers.NewEventCalendar(m.rsvp)
	}

	if m.categories != nil {
		api.CategoriesListCategoriesHandler = handlers.NewListCategories(m.categories)
	}

	if m.venues != nil {
		api.VenuesListVenuesHandler = handlers.NewListVenues(m.venues)
		api.VenuesCreateVenueHandler = handlers.NewCreateVenue(m.venues)
		api.VenuesUpdateVenueHandler = handlers.NewUpdateVenue(m.venues)
	}

	// TODO: Add more handlers as you expand the API
	// api.UsersCreateUserHandler = handlers.NewCreateUser(m.service)
	// api.UsersUpdateUserHandler = handlers.NewUpdateUser(m.service)
	// api.UsersDeleteUserHandler = handlers.NewDeleteUser(m.service)
	// api.UsersListUsersHandler = handlers.NewListUsers(m.service)

	// Build the base swagger handler.
	base := api.Serve(nil)

	// Hoist the uploads handler (may be nil when storage/files are not configured).
	var mounted http.Handler
	if m.storage != nil && m.files != nil {
		mounted = uploads.NewHandler(m.storage, m.files, m.auth.CheckAuth)
	}

	// Build the admin handler (always present; gracefully degrades when moderation
	// or events services are nil — e.g. in no-DB mode).
	adminH := admin.NewHandler(admin.Deps{
		Authenticate: m.auth.Authenticate,
		Moderation:   m.moderation,
		Events:       m.events,
		LatestReason: m.modReason,
		Organizers:   m.organizers,
		Settings:     m.settings,
	})

	// Build the organizers handler (user-facing /me/organizer + public
	// /organizers/{id}); nil in no-DB mode, in which case those paths fall to base.
	var orgH http.Handler
	if m.organizers != nil {
		orgH = organizershttp.NewHandler(organizershttp.Deps{
			Authenticate: m.auth.Authenticate,
			Organizers:   m.organizers,
			Events:       m.events,
			Store:        m.storage,
		})
	}

	// Mount the admin handler ahead of the swagger mux, then organizers, then
	// uploads, then base. These paths bypass swagger validation entirely.
	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/auth/me" || strings.HasPrefix(p, "/api/v1/admin/") {
			adminH.ServeHTTP(w, r)
			return
		}
		if orgH != nil &&
			(p == "/api/v1/me/organizer" || strings.HasPrefix(p, "/api/v1/me/organizer/") ||
				strings.HasPrefix(p, "/api/v1/organizers/")) {
			orgH.ServeHTTP(w, r)
			return
		}
		if mounted != nil &&
			(strings.HasPrefix(p, "/api/v1/uploads") || strings.HasPrefix(p, "/api/v1/files/")) {
			mounted.ServeHTTP(w, r)
			return
		}
		base.ServeHTTP(w, r)
	})

	// Build middleware chain
	handler := alice.New(
		middlewares.Recovery(),
		middlewares.Logger(),
		middlewares.Cors(m.config.CORS),
		middlewares.RateLimit(m.config.RateLimit),
	).Then(router)

	m.api = api
	m.handler = &handler

	return nil
}

// initServer initializes the HTTP server.
func (m *Module) initServer() error {
	// Create server instance
	m.server = httpserver.NewServer(m.api)

	if m.config.Host == "" {
		return fmt.Errorf("http host is required")
	}

	if m.config.Port <= 0 || m.config.Port > 65535 {
		return fmt.Errorf("http port must be between 1 and 65535")
	}

	m.server.Host = m.config.Host
	m.server.Port = m.config.Port

	// Parse and set timeouts
	timeout, err := time.ParseDuration(m.config.Timeout)
	if err != nil {
		return fmt.Errorf("parse timeout: %w", err)
	}
	m.server.ReadTimeout = timeout
	m.server.WriteTimeout = timeout

	// Set handler with middleware
	m.server.SetHandler(*m.handler)

	return nil
}
