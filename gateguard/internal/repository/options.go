package repository

import "github.com/go-pg/pg/v10/orm"

type QueryOption func(query *orm.Query)

func ApplyQueryOptions(query *orm.Query, options ...QueryOption) {
	for _, option := range options {
		option(query)
	}
}
