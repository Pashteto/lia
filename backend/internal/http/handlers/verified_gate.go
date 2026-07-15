package handlers

import (
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	apimodels "github.com/Pashteto/lia/internal/http/models"
)

const unverifiedBody = `{"code":"email_not_verified","message":"Подтвердите электронную почту, чтобы продолжить"}`

// IsVerified reports whether the principal has a verified email (nil-safe).
func IsVerified(p *apimodels.User) bool {
	return p != nil && p.EmailVerified
}

// UnverifiedResponder writes a 403 email_not_verified response.
func UnverifiedResponder() middleware.Responder {
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(unverifiedBody))
	})
}
