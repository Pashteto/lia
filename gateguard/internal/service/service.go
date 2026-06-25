package service

import (
	"time"

	sessions "github.com/andskur/gatekeeper"
	"github.com/gateway-fm/scriptorium/clog"
	"github.com/gateway-fm/scriptorium/transactions"

	"gateguard/internal/pkg/clients/organizations"
	"gateguard/internal/pkg/links"
	"gateguard/internal/pkg/notificator"
	"gateguard/internal/repository"
)

type UsersService struct {
	log                 *clog.CustomLogger
	repository          repository.IRepository
	sessions            sessions.ISessions
	extendedSession     sessions.ISessions
	notificator         notificator.INotificator
	orgs                organizations.IOrganizationsAPI
	trm                 transactions.TransactionManager
	lb                  links.LinkBuilder
	maxWeeklyInvitesNum int
	invitesTTLHours     time.Duration
}

// NewUsersService is
func NewUsersService(
	log *clog.CustomLogger,
	repo repository.IRepository,
	s sessions.ISessions,
	n notificator.INotificator,
	orgs organizations.IOrganizationsAPI,
	trm transactions.TransactionManager,
	lb links.LinkBuilder,
	maxWeeklyInvitesNum int,
	invitesTTLHours time.Duration,
) IUsersService {
	return &UsersService{
		log:                 log,
		repository:          repo,
		sessions:            s,
		extendedSession:     nil, // Will be set later
		notificator:         n,
		orgs:                orgs,
		trm:                 trm,
		lb:                  lb,
		maxWeeklyInvitesNum: maxWeeklyInvitesNum,
		invitesTTLHours:     invitesTTLHours,
	}
}

// SetExtendedSession sets an extended session for special users
func (us *UsersService) SetExtendedSession(extendedSession sessions.ISessions) {
	us.extendedSession = extendedSession
}
