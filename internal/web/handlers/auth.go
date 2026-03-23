package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "sm_session"

// Context keys injected by AuthRequired middleware.
const (
	ctxKeyUserDB   = "sm_user_db"
	ctxKeyUsername = "sm_username"
	ctxKeyIsAdmin  = "sm_is_admin"
)

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// AuthRequired is a Gin middleware that ensures the caller is authenticated.
// It accepts both a session cookie (browser login) and an
// "Authorization: Bearer <api-key>" header (programmatic access).
func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- Bearer token (API key) ---
		if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			plainToken := strings.TrimPrefix(authHeader, "Bearer ")
			username, err := h.apiKeyStore().Verify(plainToken)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
				return
			}
			db, err := h.registry.Get(username)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}
			u, err := h.users.GetByUsername(username)
			if err != nil || u == nil || u.Disabled {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}
			c.Set(ctxKeyUserDB, db)
			c.Set(ctxKeyUsername, username)
			c.Set(ctxKeyIsAdmin, u.IsAdmin)
			c.Next()
			return
		}

		// --- Session cookie ---
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil || !h.validSession(cookie) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		username, _ := verifyToken(cookie, h.cfg.SecretKey)
		db, err := h.registry.Get(username)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.gohtml", gin.H{"Error": "failed to open user database"})
			c.Abort()
			return
		}
		u, err := h.users.GetByUsername(username)
		if err != nil || u == nil || u.Disabled {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set(ctxKeyUserDB, db)
		c.Set(ctxKeyUsername, username)
		c.Set(ctxKeyIsAdmin, u.IsAdmin)
		c.Next()
	}
}

// AdminRequired is a Gin middleware that allows only admin users.
// Must be used after AuthRequired.
func (h *Handler) AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool(ctxKeyIsAdmin) {
			c.HTML(http.StatusForbidden, "error.gohtml", gin.H{"Error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// isAdmin reports whether the current request is from an admin user.
func (h *Handler) isAdmin(c *gin.Context) bool {
	return c.GetBool(ctxKeyIsAdmin)
}

func (h *Handler) validSession(token string) bool {
	username, ok := verifyToken(token, h.cfg.SecretKey)
	if !ok {
		return false
	}
	u, err := h.users.GetByUsername(username)
	return err == nil && u != nil && !u.Disabled
}

// ---------------------------------------------------------------------------
// Auth handlers
// ---------------------------------------------------------------------------

// LoginPage renders the login form.
// When no users exist yet, a notice is shown directing the operator to the
// console-printed setup URL.
func (h *Handler) LoginPage(c *gin.Context) {
	count, _ := h.users.Count()
	c.HTML(http.StatusOK, "login.gohtml", gin.H{
		"Error":     "",
		"SetupMode": count == 0,
	})
}

// LoginSubmit validates credentials and sets a session cookie.
// If the user has 2FA enabled, a short-lived pending cookie is issued instead
// and the browser is redirected to the TOTP verification step.
func (h *Handler) LoginSubmit(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	u, err := h.users.GetByUsername(username)
	if err != nil || u == nil {
		c.HTML(http.StatusUnauthorized, "login.gohtml", gin.H{"Error": "Invalid username or password"})
		return
	}

	if u.Disabled {
		c.HTML(http.StatusUnauthorized, "login.gohtml", gin.H{"Error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.gohtml", gin.H{"Error": "Invalid username or password"})
		return
	}

	// If 2FA is enabled, issue a short-lived pending token and redirect to the TOTP step.
	_, enabled, _ := h.users.GetTOTP(username)
	if enabled {
		pendingToken := signPendingToken(username, h.cfg.SecretKey)
		c.SetCookie("sm_pending", pendingToken, int(5*time.Minute/time.Second), "/", "", false, true)
		c.Redirect(http.StatusFound, "/login/2fa")
		return
	}

	token := signToken(username, h.cfg.SecretKey)
	c.SetCookie(sessionCookieName, token, int(24*time.Hour/time.Second), "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// TwoFALoginPage renders the TOTP verification step of the login flow.
func (h *Handler) TwoFALoginPage(c *gin.Context) {
	pending, err := c.Cookie("sm_pending")
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	_, ok := verifyPendingToken(pending, h.cfg.SecretKey)
	if !ok {
		c.SetCookie("sm_pending", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
		return
	}
	c.HTML(http.StatusOK, "login_2fa.gohtml", gin.H{"Error": ""})
}

// TwoFALoginSubmit validates the TOTP code and completes the login.
func (h *Handler) TwoFALoginSubmit(c *gin.Context) {
	pending, err := c.Cookie("sm_pending")
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	username, ok := verifyPendingToken(pending, h.cfg.SecretKey)
	if !ok {
		c.SetCookie("sm_pending", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	code := strings.TrimSpace(c.PostForm("code"))
	secret, enabled, totpErr := h.users.GetTOTP(username)
	if totpErr != nil || !enabled || secret == "" {
		c.SetCookie("sm_pending", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if !totp.Validate(code, secret) {
		c.HTML(http.StatusUnauthorized, "login_2fa.gohtml", gin.H{"Error": "Invalid code. Please try again."})
		return
	}

	c.SetCookie("sm_pending", "", -1, "/", "", false, true)
	token := signToken(username, h.cfg.SecretKey)
	c.SetCookie(sessionCookieName, token, int(24*time.Hour/time.Second), "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// Logout clears the session cookie.
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}
