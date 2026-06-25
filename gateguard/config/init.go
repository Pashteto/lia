package config

import (
	"log/slog"

	"github.com/spf13/viper"
)

type Env string

const (
	EnvLocal Env = "local"
	EnvDev   Env = "dev"
	EnvProd  Env = "prod"
)

// init initialize default config params
func init() {
	// environment - could be "local", "prod", "dev"
	viper.SetDefault("env", EnvProd)
	viper.SetDefault("invites.maxweeklyinvitesnum", 20)
	viper.SetDefault("invites.ttlhours", "168h")

	viper.SetDefault("referralLinkFormat", `https://api.presto.dev.gateway.fm/api/v1/auth/signin/google?redirect=presto.dev.gateway.fm&refCode=%s`)

	// grpc server
	viper.SetDefault("grpc.port", 9090)
	viper.SetDefault("grpc.timeout", "120s")

	// db postgresql
	viper.SetDefault("db.address", "postgres://presto:presto@localhost:54322/presto?sslmode=disable")

	// auth
	viper.SetDefault("auth.secret", "prestopresto")
	viper.SetDefault("auth.expire", "72h")

	// notificator
	viper.SetDefault("notificator.username", "infra@gateway.fm")
	viper.SetDefault("notificator.password", "")
	viper.SetDefault("notificator.from", "sales@gateway.fm")
	viper.SetDefault("notificator.address", "smtp.gmail.com:587")
	viper.SetDefault("notificator.organization", "https://jade-bitter-partridge-776.mypinata.cloud/ipfs/QmeTcCmVvQgEa6tKFAZpASEAcXe3tr3LGNxAoqedFvzhV1")

	// orgs client
	viper.SetDefault("organizations.address", "127.0.0.1:9097")
	viper.SetDefault("organizations.timeout", "10s")

	// key value storage
	viper.SetDefault("redis.address", "redis://localhost:6379")

	// logging level
	viper.SetDefault("log.level", slog.LevelDebug)
}
