package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const userIDKey contextKey = "userID"

type Session struct {
	UserID    int64
	ExpiresAt time.Time
	Script    string // "latin", "cyrillic", "both"
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) Create(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	s.mu.Lock()
	s.sessions[token] = &Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Script:    "both",
	}
	s.mu.Unlock()

	return token, nil
}

func (s *SessionStore) Get(token string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil
	}
	return sess
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *SessionStore) SetScript(token, mode string) {
	s.mu.Lock()
	if sess, ok := s.sessions[token]; ok {
		sess.Script = mode
	}
	s.mu.Unlock()
}

func (s *SessionStore) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		sess := s.Get(cookie.Value)
		if sess == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r.Context()) == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func GetUserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}
