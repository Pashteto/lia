package handlers

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	categoriesdomain "github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	categoriesops "github.com/Pashteto/lia/internal/http/server/operations/categories"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListCategories handler returns the curated category taxonomy.
type ListCategories struct {
	categories categoriesdomain.Service
}

// NewListCategories creates a ListCategories handler.
func NewListCategories(svc categoriesdomain.Service) *ListCategories {
	return &ListCategories{categories: svc}
}

// Handle GET /categories.
func (h *ListCategories) Handle(params categoriesops.ListCategoriesParams) middleware.Responder {
	list, err := h.categories.List(params.HTTPRequest.Context())
	if err != nil {
		logger.Log().Errorf("list categories: %s", err.Error())
		return categoriesops.NewListCategoriesServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, err, nil))
	}

	payload := make([]*apimodels.Category, 0, len(list))
	for _, c := range list {
		payload = append(payload, formatter.CategoryToAPI(c))
	}

	return categoriesops.NewListCategoriesOK().WithPayload(payload)
}
