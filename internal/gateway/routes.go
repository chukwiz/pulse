package gateway

import (
	"net/http"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /v1/events", h.createEvent)

	mux.HandleFunc("GET /v1/events", h.listEvents)

	mux.HandleFunc("GET /v1/events/{id}", h.getEvent)
}
