package templates

import "context"

//go:generate ../../../../bin/mockery --name ITemplate

type ITemplate interface {
	GetTemplateAsString(ctx context.Context) (string, error)
	TemplateName() string
	Subject() string
}
