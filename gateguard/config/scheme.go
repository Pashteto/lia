package config

import "time"

// Scheme represents the application configuration scheme.
type Scheme struct {
	// Env is the application environment.
	Env                Env
	ReferralLinkFormat string
	Invites            *Invites
	Grpc               *Server
	Db                 *Db
	Redis              *Redis
	Auth               *Auth
	Notificator        *Notificator
	Organizations      *Client
	Log                *Log
}

type Invites struct {
	MaxWeeklyInvitesNum int
	TTLHours            time.Duration
}

// Server represent basic server params
type Server struct {
	Port    int
	Timeout string
}

// Auth config Scheme for Auth params
type Auth struct {
	Secret string
	Expire string
}

// Db is service Data base connection params
type Db struct {
	Address string
}

// Redis represent Redis connection scheme
type Redis struct {
	Address string
}

// Notificator holds notification service settings.
type Notificator struct {
	Username     string
	Password     string
	From         string
	Address      string
	Organization string
}

type Client struct {
	Address string
	Timeout time.Duration
}

// Log contains logging configuration.
type Log struct {
	Level int
}
