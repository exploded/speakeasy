package handlers

import (
	"net/http"

	"speakeasy/internal/db"
	"speakeasy/internal/middleware"
)

const adminUsername = "james"

type AdminHandler struct {
	queries *db.Queries
	tmpl    *TemplateRenderer
}

func NewAdminHandler(q *db.Queries, t *TemplateRenderer) *AdminHandler {
	return &AdminHandler{queries: q, tmpl: t}
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil || user.Username != adminUsername {
		http.NotFound(w, r)
		return
	}

	users, err := h.queries.ListAllUsers(r.Context())
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	h.tmpl.Render(w, "admin_users.html", map[string]interface{}{
		"Title": "All Users",
		"User":  user,
		"Users": users,
	})
}
