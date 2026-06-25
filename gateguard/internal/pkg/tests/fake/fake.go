package fake

import (
	"math/rand"
	"net"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/models"
	omodels "gateguard/internal/pkg/clients/organizations/models"
	"gateguard/internal/pkg/converters/ipconv"
)

// GenerateRandomIP generates a random IPv4 address using a local random source.
func GenerateRandomIP() string {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	ip := net.IPv4(byte(rnd.Intn(256)), byte(rnd.Intn(256)), byte(rnd.Intn(256)), byte(rnd.Intn(256)))
	return ip.String()
}

func User() *models.User {
	userID := uuid.Must(uuid.NewV7())
	userIP, err := ipconv.IpToUint32(GenerateRandomIP())
	if err != nil {
		userIP = 0
	}

	return &models.User{
		UUID:      userID,
		Email:     gofakeit.Email(),
		Name:      gofakeit.Name(),
		Avatar:    gofakeit.HipsterWord(),
		Status:    models.UserActive,
		StatusSQL: models.UserActive.String(),
		IP:        userIP,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
		DeletedAt: time.Time{},
	}
}

func Invitation() *models.Invitation {
	inv := &models.Invitation{
		Inviter:      gofakeit.Email(),
		Organization: pointer.Ref(uuid.Must(uuid.NewV4())),
		Invitee:      gofakeit.Email(),
		InviteeRole:  oneOf([]omodels.RoleType{omodels.RoleCommon, omodels.RoleCreator, omodels.RoleBilling, omodels.RoleOwner}).String(),
		Status:       models.Pending,
		CreatedAt:    time.Now(),
	}

	inv.ReferralCode = inv.GenerateReferralCode()

	return inv
}

// oneOf selects a random element from a list.
func oneOf[T any](list []T) T {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return list[r.Intn(len(list))]
}
