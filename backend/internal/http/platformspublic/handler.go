// Package platformspublic serves the public read-only whitelist:
// GET /api/v1/trusted-platforms. Mounted ahead of the swagger mux.
package platformspublic

import (
	"encoding/json"
	"net/http"

	"github.com/Pashteto/lia/internal/platforms"
	"github.com/Pashteto/lia/pkg/logger"
)

// Deps are the handler's injected dependencies.
type Deps struct{ Platforms platforms.Service }

type handler struct{ deps Deps }

// NewHandler builds the /api/v1/trusted-platforms handler. No auth — this
// list is public (organizers need it while composing an external event, and
// it carries no sensitive data).
func NewHandler(deps Deps) http.Handler { return &handler{deps: deps} }

type row struct {
	DomainSuffix string `json:"domain_suffix"`
	DisplayName  string `json:"display_name"`
	Category     string `json:"category"`
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows, err := h.deps.Platforms.ListActive(r.Context())
	if err != nil {
		logger.Log().Errorf("list trusted platforms: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	out := make([]row, 0, len(rows))
	for _, p := range rows {
		out = append(out, row{DomainSuffix: p.DomainSuffix, DisplayName: p.DisplayName, Category: p.Category})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
