package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// Session Store
// ============================================================

type Session struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IP        string    `json:"ip"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // token -> session
}

var sessionStore = &SessionStore{
	sessions: make(map[string]*Session),
}

func (s *SessionStore) Create(username, ip string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := generateToken()
	now := time.Now()
	session := &Session{
		Token:     token,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		IP:        ip,
	}
	s.sessions[token] = session
	return token
}

func (s *SessionStore) Get(token string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok {
		return nil
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		return nil
	}
	return session
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// ============================================================
// Rate Limiter
// ============================================================

type LoginAttempt struct {
	Count     int
	FirstAt   time.Time
	LockedAt  time.Time
	Locked    bool
}

type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*LoginAttempt // IP -> attempt
}

var rateLimiter = &RateLimiter{
	attempts: make(map[string]*LoginAttempt),
}

const (
	maxAttempts     = 5
	lockoutDuration = 15 * time.Minute
	windowDuration  = 5 * time.Minute
)

func (rl *RateLimiter) Allow(ip string) (bool, string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	attempt, ok := rl.attempts[ip]
	if !ok {
		rl.attempts[ip] = &LoginAttempt{Count: 1, FirstAt: time.Now()}
		return true, ""
	}

	// Check if currently locked out
	if attempt.Locked {
		if time.Since(attempt.LockedAt) > lockoutDuration {
			// Unlock after lockout period
			attempt.Locked = false
			attempt.Count = 1
			attempt.FirstAt = time.Now()
			return true, ""
		}
		remaining := lockoutDuration - time.Since(attempt.LockedAt)
		return false, fmt.Sprintf("Too many attempts. Try again in %d seconds", int(remaining.Seconds()))
	}

	// Reset if window expired
	if time.Since(attempt.FirstAt) > windowDuration {
		attempt.Count = 1
		attempt.FirstAt = time.Now()
		return true, ""
	}

	attempt.Count++
	if attempt.Count > maxAttempts {
		attempt.Locked = true
		attempt.LockedAt = time.Now()
		return false, "Too many failed attempts. Locked out for 15 minutes."
	}

	return true, ""
}

// ============================================================
// CAPTCHA (simple math)
// ============================================================

type CaptchaQuestion struct {
	ID      string
	A       int
	B       int
	Op      string // "+", "-"
	Answer  int
	Expires time.Time
}

type CaptchaStore struct {
	mu       sync.RWMutex
	questions map[string]*CaptchaQuestion
}

var captchaStore = &CaptchaStore{
	questions: make(map[string]*CaptchaQuestion),
}

func (cs *CaptchaStore) Generate() *CaptchaQuestion {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	id := generateToken()
	a := randInt(1, 50)
	b := randInt(1, 50)
	op := "+"
	answer := a + b

	// Sometimes use subtraction
	if randInt(0, 1) == 1 && a > b {
		op = "-"
		answer = a - b
	}

	q := &CaptchaQuestion{
		ID:      id,
		A:       a,
		B:       b,
		Op:      op,
		Answer:  answer,
		Expires: time.Now().Add(5 * time.Minute),
	}
	cs.questions[id] = q

	// Cleanup old
	for k, v := range cs.questions {
		if time.Now().After(v.Expires) {
			delete(cs.questions, k)
		}
	}

	return q
}

func (cs *CaptchaStore) Verify(id string, answer int) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	q, ok := cs.questions[id]
	if !ok {
		return false
	}
	if time.Now().After(q.Expires) {
		delete(cs.questions, id)
		return false
	}

	result := q.Answer == answer
	delete(cs.questions, id) // one-time use
	return result
}

// ============================================================
// Admin Credentials
// ============================================================

type AdminUser struct {
	Username       string
	PasswordHash   string
	CreatedAt      time.Time
}

var adminUser *AdminUser

func setupAdmin(username, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[AUTH] Failed to hash password: %v", err)
	}

	adminUser = &AdminUser{
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	log.Printf("[AUTH] Admin user '%s' configured (bcrypt hash: %d chars)", username, len(hash))
}

func verifyPassword(password string) bool {
	if adminUser == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(password))
	return err == nil
}

// ============================================================
// Session Middleware
// ============================================================

func sessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Agent API uses X-API-Key (skip session check)
		if strings.HasPrefix(r.URL.Path, "/api/agent/") {
			key := r.Header.Get("X-API-Key")
			if key != "" && key == config.APIKey {
				next(w, r)
				return
			}
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Health check (no auth needed)
		if r.URL.Path == "/health" {
			next(w, r)
			return
		}

		// Login page and login API (no session needed)
		if r.URL.Path == "/login" || r.URL.Path == "/api/login" || r.URL.Path == "/api/captcha" {
			next(w, r)
			return
		}

		// Dashboard and admin API — check session
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			// No session — show login
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session := sessionStore.Get(cookie.Value)
		if session == nil {
			// Invalid/expired session
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next(w, r)
	}
}

// ============================================================
// Login Handler
// ============================================================

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getIP(r)

	// Rate limit check
	allowed, msg := rateLimiter.Allow(ip)
	if !allowed {
		jsonError(w, msg, http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		CaptchaID  string `json:"captcha_id"`
		CaptchaAns int    `json:"captcha_answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Verify CAPTCHA
	if !captchaStore.Verify(req.CaptchaID, req.CaptchaAns) {
		jsonError(w, "invalid captcha answer", http.StatusBadRequest)
		return
	}

	// Verify credentials
	if req.Username != adminUser.Username || !verifyPassword(req.Password) {
		log.Printf("[AUTH] Failed login attempt for '%s' from %s", req.Username, ip)
		jsonError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create session
	token := sessionStore.Create(req.Username, ip)
	log.Printf("[AUTH] Successful login for '%s' from %s", req.Username, ip)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"message": "login successful"}})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		sessionStore.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"message": "logged out"}})
}

func handleCaptcha(w http.ResponseWriter, r *http.Request) {
	q := captchaStore.Generate()
	jsonResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"captcha_id": q.ID,
			"question":   fmt.Sprintf("%d %s %d = ?", q.A, q.Op, q.B),
		},
	})
}

func handleCheckSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		jsonResponse(w, APIResponse{Success: false, Data: map[string]bool{"authenticated": false}})
		return
	}

	session := sessionStore.Get(cookie.Value)
	if session == nil {
		jsonResponse(w, APIResponse{Success: false, Data: map[string]bool{"authenticated": false}})
		return
	}

	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{
		"authenticated": true,
		"username":      session.Username,
	}})
}

// ============================================================
// Helpers
// ============================================================

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randInt(min, max int) int {
	b := make([]byte, 4)
	rand.Read(b)
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 {
		n = -n
	}
	return min + (n % (max - min + 1))
}

func getIP(r *http.Request) string {
	// Check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
