package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	sessions "github.com/andskur/gatekeeper"
	"github.com/andskur/gatekeeper/jwt"
	"github.com/andskur/gatekeeper/persistance"
	nosql "github.com/andskur/gatekeeper/persistance/redis"
	"github.com/gateway-fm/scriptorium/clog"
	"github.com/gateway-fm/scriptorium/transactions"
	version "github.com/gateway-fm/scriptorium/versioner"
	"github.com/go-pg/pg/v10"
	"github.com/redis/go-redis/v9"

	"gateguard/config"
	"gateguard/internal/pkg/clients/organizations"
	"gateguard/internal/pkg/links"
	"gateguard/internal/pkg/notificator"
	"gateguard/internal/repository"
	"gateguard/internal/server"
	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

// App is main microservice application instance that
// have all necessary dependencies inside structure
type App struct {
	ctx             context.Context
	log             *clog.CustomLogger
	config          *config.Scheme
	version         *version.Version
	grpcServer      server.IGrpc
	db              *pg.DB
	keyValue        persistance.IStorage
	session         sessions.ISessions
	extendedSession sessions.ISessions
	srv             service.IUsersService
	mailer          notificator.INotificator
	orgs            organizations.IOrganizationsAPI
	repository      repository.IRepository
	trf             *transactions.PgTransactionFactory
	trm             *transactions.PgTransactionManager
	lb              links.LinkBuilder
}

// NewApplication create new App instance
func NewApplication() (app *App, err error) {
	ver, err := version.NewVersion()
	if err != nil {
		return nil, fmt.Errorf("init app version: %w", err)
	}

	return &App{
		ctx:     context.Background(),
		config:  &config.Scheme{},
		version: ver,
	}, nil
}

// Init initialize application and all necessary instances
func (app *App) Init() error {
	app.log = clog.NewCustomLogger(os.Stdout, clog.Level(app.Config().Log.Level), false)

	if err := app.initDb(app.config.Db); err != nil {
		return fmt.Errorf("application database initialisation: %w", err)
	}

	if err := app.initKeyValue(app.config.Redis); err != nil {
		return fmt.Errorf("application keyValue storage initialisation: %w", err)
	}
	if err := app.initSessions(app.config.Auth); err != nil {
		return fmt.Errorf("application session instance initialisation: %w", err)
	}

	if err := app.initService(); err != nil {
		return fmt.Errorf("could not init services: %w", err)
	}

	if err := app.initGrpc(app.config.Grpc); err != nil {
		return fmt.Errorf("application grpc server initialisation: %w", err)
	}

	return nil
}

// initDb initialize Application database instance
func (app *App) initDb(cfg *config.Db) (err error) {
	opts, err := pg.ParseURL(cfg.Address)
	if err != nil {
		return err
	}

	app.log.Info(fmt.Sprintf("Connecting to Postgresql database %s on %s...", opts.Database, opts.Addr))

	app.db = pg.Connect(opts)

	if _, err = app.db.Exec("SELECT 1"); err != nil {
		return err
	}

	app.trf = transactions.NewPgTransactionFactory(app.db)
	app.trm = transactions.NewPgTransactionManager(app.trf, transactions.Options{AlwaysRollback: false})
	app.repository = repository.NewRepository(app.trf)

	app.log.Info(fmt.Sprintf("Connected to Postgresql database %s on %s", opts.Database, opts.Addr))
	return nil
}

// Expire expires invitations that have been ignored for the TTL period that is set in the config
func (app *App) Expire() error {
	app.log = clog.NewCustomLogger(os.Stdout, clog.Level(app.Config().Log.Level), false)

	if err := app.initDb(app.config.Db); err != nil {
		return fmt.Errorf("application database initialisation: %w", err)
	}

	if err := app.initService(); err != nil {
		return fmt.Errorf("could not init services: %w", err)
	}

	return app.srv.ExpireInvitations(app.ctx)
}

// initKeyValue initialize Application Key-Value persistence storage
func (app *App) initKeyValue(cfg *config.Redis) (err error) {
	opts, err := redis.ParseURL(cfg.Address)
	if err != nil {
		return fmt.Errorf("initKeyValue: parseURL: %w", err)
	}

	opts.DialTimeout = 10 * time.Second
	opts.ReadTimeout = 30 * time.Second
	opts.WriteTimeout = 30 * time.Second
	opts.PoolSize = 10
	opts.PoolTimeout = 30 * time.Second

	app.keyValue = nosql.New(opts)

	if _, err = app.keyValue.Ping(app.ctx); err != nil {
		return fmt.Errorf("init KeyValue: %w", err)
	}

	app.log.Info(fmt.Sprintf("Connected to Redis key-value database on %s", opts.Addr))
	return
}

// initSessions initialize session layer
func (app *App) initSessions(cfg *config.Auth) error {
	expire, err := time.ParseDuration(cfg.Expire)
	if err != nil {
		return fmt.Errorf("parse session expire value: %w", err)
	}

	app.log.Info(fmt.Sprintf("Initializing regular session with expire: %s", expire))

	// Create regular session (3 days)
	session, err := jwt.NewJwtSession([]byte(cfg.Secret), expire, app.keyValue)
	if err != nil {
		return fmt.Errorf("create jwt-based session: %w", err)
	}

	// Create extended session for special users (3 months)
	extendedExpire := 3 * 30 * 24 * time.Hour // 3 months approximation
	app.log.Info(fmt.Sprintf("Initializing extended session with expire: %s", extendedExpire))

	extendedSession, err := jwt.NewJwtSession([]byte(cfg.Secret), extendedExpire, app.keyValue)
	if err != nil {
		return fmt.Errorf("create extended jwt-based session: %w", err)
	}

	app.session = session
	app.extendedSession = extendedSession
	app.log.Info("sessions initialized")
	return nil
}

// initService initialize Application service layer
func (app *App) initService() error {
	app.mailer = notificator.NewSMTPNotificator(
		app.Config().Notificator.Username,
		app.Config().Notificator.Password,
		app.Config().Notificator.From,
		app.Config().Notificator.Address,
		app.Config().Notificator.Organization,
		app.log,
	)

	app.lb = links.NewLinkBuilder(app.config.ReferralLinkFormat)

	if err := app.initOrganizationsApi(app.config.Organizations); err != nil {
		return err
	}

	log.Println(app.config.Invites.MaxWeeklyInvitesNum)

	app.srv = service.NewUsersService(
		app.log,
		app.repository,
		app.session,
		app.mailer,
		app.orgs,
		app.trm,
		app.lb,
		app.config.Invites.MaxWeeklyInvitesNum,
		app.config.Invites.TTLHours,
	)

	// Set extended session for special users
	app.log.Info("Setting extended session for service")
	app.srv.SetExtendedSession(app.extendedSession)

	app.log.Info("application service layer initialized")

	return nil
}

// initGrpc initialize Application GRPC server instance
func (app *App) initGrpc(cfg *config.Server) (err error) {
	app.grpcServer, err = server.NewServer(cfg, app.log)
	if err != nil {
		return fmt.Errorf("create new GRPC server instance: %w", err)
	}

	proto.RegisterGateguardServiceServer(app.grpcServer.GetServer(), server.NewGateguardHandlers(app.srv, app.log))

	app.log.Info(fmt.Sprintf("App GRPC server initialized on %d", cfg.Port))
	return nil
}

// Serve start serving Application service
func (app *App) Serve() error {
	go app.grpcServer.Listen()

	// Gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-quit

	return nil
}

// initOrganizationsApi initialize Application Organizations API instance
func (app *App) initOrganizationsApi(cfg *config.Client) error {
	var err error
	app.orgs, err = organizations.New(cfg.Address, app.log, cfg.Timeout)
	if err != nil {
		return err
	}

	app.log.Info(fmt.Sprintf("Application Organizations API initialized on %s", cfg.Address))
	return nil
}

// Stop shutdown the application
func (app *App) Stop() error {
	app.grpcServer.Close()

	if err := app.keyValue.Close(); err != nil {
		return fmt.Errorf("close redis connection: %w", err)
	}

	return app.db.Close()
}

// Config return App config Scheme
func (app *App) Config() *config.Scheme {
	return app.config
}

// Version return application current version
func (app *App) Version() string {
	return app.version.String()
}

// CreateAddr creates address string from host and port
func CreateAddr(host string, port int) string {
	return fmt.Sprintf("%s:%v", host, port)
}
