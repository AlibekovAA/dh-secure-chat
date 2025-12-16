package http

import (
	"net/http"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func HealthHandler(log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		log.Infof("health check request")
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
