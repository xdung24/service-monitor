package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/png"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/xdung24/conductor/internal/mailer"
	"golang.org/x/crypto/bcrypt"
)

// SecurityPage renders the combined security page (change password + 2FA).
func (h *Handler) SecurityPage(c *gin.Context) {
	username := h.username(c)
	_, twoFAEnabled, _ := h.users.GetTOTP(username)
	flash, _ := c.Cookie("sm_flash")
	if flash != "" {
		c.SetCookie("sm_flash", "", -1, "/", "", false, true)
	}
	c.HTML(http.StatusOK, "account_security.gohtml", h.pageData(c, gin.H{
		"TwoFAEnabled": twoFAEnabled,
		"Flash":        flash,
		"PwError":      "",
		"TwoFAError":   "",
	}))
}

// TwoFASetupPage generates a new TOTP key, stores the pending secret, and
// renders the QR code so the user can scan it with their authenticator app.
func (h *Handler) TwoFASetupPage(c *gin.Context) {
	username := h.username(c)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "ServiceConductor",
		AccountName: username,
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "Failed to generate 2FA key. Please try again.",
		}))
		return
	}

	if err := h.users.SetTOTPSecret(username, key.Secret()); err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "Failed to save 2FA key. Please try again.",
		}))
		return
	}

	// Encode QR code image as a base64 data URI so it can be embedded directly
	// in the HTML without a separate image endpoint.
	qrImg, err := key.Image(200, 200)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false, "Flash": "", "PwError": "", "TwoFAError": "Failed to generate QR code.",
		}))
		return
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrImg); err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false, "Flash": "", "PwError": "", "TwoFAError": "Failed to encode QR code.",
		}))
		return
	}
	// #nosec G203 -- data URI is entirely server-generated base64 PNG; no user input is interpolated.
	qrDataURI := template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()))

	c.HTML(http.StatusOK, "account_security.gohtml", h.pageData(c, gin.H{
		"TwoFAEnabled": false,
		"SetupMode":    true,
		"QRDataURI":    qrDataURI,
		"TOTPSecret":   key.Secret(),
		"Flash":        "",
		"PwError":      "",
		"TwoFAError":   "",
	}))
}

// TwoFAVerify validates the TOTP code entered by the user after scanning the QR
// code and, on success, marks 2FA as enabled.
func (h *Handler) TwoFAVerify(c *gin.Context) {
	username := h.username(c)
	code := c.PostForm("code")

	secret, _, err := h.users.GetTOTP(username)
	if err != nil || secret == "" {
		c.HTML(http.StatusBadRequest, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "No pending 2FA setup found. Please start the setup again.",
		}))
		return
	}

	if !totp.Validate(code, secret) {
		// Re-render the setup page with the same QR code so the user can retry.
		keyURL := fmt.Sprintf("otpauth://totp/ServiceConductor%%3A%s?secret=%s&issuer=ServiceConductor",
			url.QueryEscape(username), secret)
		key, parseErr := otp.NewKeyFromURL(keyURL)

		var qrDataURI template.URL
		if parseErr == nil {
			if qrImg, imgErr := key.Image(200, 200); imgErr == nil {
				var buf bytes.Buffer
				if pngErr := png.Encode(&buf, qrImg); pngErr == nil {
					// #nosec G203 -- data URI is entirely server-generated base64 PNG; no user input is interpolated.
					qrDataURI = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()))
				}
			}
		}

		c.HTML(http.StatusUnauthorized, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false,
			"SetupMode":    true,
			"QRDataURI":    qrDataURI,
			"TOTPSecret":   secret,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "Invalid code. Please try again.",
		}))
		return
	}

	if err := h.users.EnableTOTP(username); err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": false,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "Failed to enable 2FA. Please try again.",
		}))
		return
	}

	// Notify the user that 2FA was enabled (fire-and-forget).
	h.mailer.SendAsync(username, "Two-factor authentication enabled", mailer.RenderTwoFAEnabled())

	c.SetCookie("sm_flash", "Two-factor authentication has been enabled.", 60, "/", "", false, true)
	c.Redirect(http.StatusFound, "/account/security")
}

// TwoFADisable disables TOTP for the current user and clears the stored secret.
func (h *Handler) TwoFADisable(c *gin.Context) {
	username := h.username(c)
	if err := h.users.DisableTOTP(username); err != nil {
		c.HTML(http.StatusInternalServerError, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": true,
			"Flash":        "",
			"PwError":      "",
			"TwoFAError":   "Failed to disable 2FA. Please try again.",
		}))
		return
	}
	c.SetCookie("sm_flash", "Two-factor authentication has been disabled.", 60, "/", "", false, true)
	c.Redirect(http.StatusFound, "/account/security")
}

// AccountChangePassword handles the self-service change-password form submission.
// The user must supply their current password to authorise the change.
func (h *Handler) AccountChangePassword(c *gin.Context) {
	username := h.username(c)
	current := c.PostForm("current_password")
	password := c.PostForm("password")
	confirm := c.PostForm("confirm_password")

	renderErr := func(msg string) {
		_, twoFAEnabled, _ := h.users.GetTOTP(username)
		c.HTML(http.StatusBadRequest, "account_security.gohtml", h.pageData(c, gin.H{
			"TwoFAEnabled": twoFAEnabled,
			"Flash":        "",
			"PwError":      msg,
			"TwoFAError":   "",
		}))
	}

	if current == "" || password == "" || confirm == "" {
		renderErr("All fields are required")
		return
	}
	if len(password) < 8 {
		renderErr("New password must be at least 8 characters")
		return
	}
	if password != confirm {
		renderErr("New passwords do not match")
		return
	}

	u, _ := h.users.GetByUsername(username)
	if u == nil {
		renderErr("User not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(current)); err != nil {
		renderErr("Current password is incorrect")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		renderErr("Internal error")
		return
	}

	if err := h.users.UpdatePassword(username, string(hashed)); err != nil {
		renderErr("Failed to update password: " + err.Error())
		return
	}

	h.mailer.SendAsync(username, "Your password has been changed", mailer.RenderPasswordChangedByReset())
	c.SetCookie("sm_flash", "Password updated successfully.", 60, "/", "", false, true)
	c.Redirect(http.StatusFound, "/account/security")
}
