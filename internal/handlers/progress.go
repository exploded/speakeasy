package handlers

import (
	"net/http"

	"speakeasy/internal/middleware"
)

type ProgressHandler struct {
	sessions *middleware.SessionStore
}

func NewProgressHandler(s *middleware.SessionStore) *ProgressHandler {
	return &ProgressHandler{sessions: s}
}

func (h *ProgressHandler) SetScriptPreference(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mode := r.FormValue("mode")
	if mode != "latin" && mode != "cyrillic" && mode != "both" {
		mode = "both"
	}

	h.sessions.SetScript(cookie.Value, mode)
	w.WriteHeader(http.StatusOK)
}
