package repository

import (
	"fmt"

	"github.com/go-pg/pg/v10/orm"

	"gateguard/internal/models"
)

// UserGetter represent Fortune Users available Getters
type UserGetter int

// User getter constants
const (
	UserUUID UserGetter = iota
	Email
)

// userGetters is slice of User Getters string representations
var userGetters = [...]string{
	UserUUID: "uuid",
	Email:    "email",
}

// String return UserGetter enum as a string
func (g UserGetter) String() string {
	return userGetters[g]
}

// Get is
func (g UserGetter) Get(query *orm.Query, model *models.User) error {
	switch g {
	case UserUUID:
		query.WherePK()
	case Email:
		query.Where(fmt.Sprintf("%s =?", g.String()), model.Email)
	default:
		return fmt.Errorf("unsupproted user getter: %s", g)
	}
	return nil
}
