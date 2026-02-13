package handlers

import (
	"net/http"

	"speakeasy/internal/db"
	"speakeasy/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	queries  *db.Queries
	sessions *middleware.SessionStore
	tmpl     *TemplateRenderer
}

func NewAuthHandler(q *db.Queries, s *middleware.SessionStore, t *TemplateRenderer) *AuthHandler {
	return &AuthHandler{queries: q, sessions: s, tmpl: t}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "login.html", map[string]interface{}{
		"IsRegister": false,
	})
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "login.html", map[string]interface{}{
		"IsRegister": true,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		h.tmpl.Render(w, "login.html", map[string]interface{}{
			"IsRegister": false,
			"Error":      "Invalid username or password",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.tmpl.Render(w, "login.html", map[string]interface{}{
			"IsRegister": false,
			"Error":      "Invalid username or password",
		})
		return
	}

	token, err := h.sessions.Create(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	displayName := r.FormValue("display_name")
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if displayName == "" || username == "" || email == "" || password == "" {
		h.tmpl.Render(w, "login.html", map[string]interface{}{
			"IsRegister": true,
			"Error":      "All fields are required",
		})
		return
	}

	if len(password) < 6 {
		h.tmpl.Render(w, "login.html", map[string]interface{}{
			"IsRegister": true,
			"Error":      "Password must be at least 6 characters",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
	})
	if err != nil {
		h.tmpl.Render(w, "login.html", map[string]interface{}{
			"IsRegister": true,
			"Error":      "Username or email already taken",
		})
		return
	}

	// Initialize lesson progress - first lesson available, rest locked
	initLessonProgress(r.Context(), h.queries, user.ID)

	token, err := h.sessions.Create(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
