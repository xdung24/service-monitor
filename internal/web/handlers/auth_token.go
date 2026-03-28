package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// signToken creates a simple HMAC-signed token: base64("username:iatUnix").base64(hmac).
func signToken(username, secret string) string {
	return signTokenWithIAT(username, secret, time.Now().UTC().Unix())
}

func signTokenWithIAT(username, secret string, iatUnix int64) string {
	payload := fmt.Sprintf("%s:%d", username, iatUnix)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + sig
}

// verifyToken validates the token and returns the username if valid.
func verifyToken(token, secret string) (string, bool) {
	username, _, ok := verifyTokenWithIAT(token, secret)
	return username, ok
}

// verifyTokenWithIAT validates token and returns username + issued-at unix timestamp.
func verifyTokenWithIAT(token, secret string) (string, int64, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", 0, false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", 0, false
	}
	payload := string(payloadBytes)
	idx := strings.LastIndex(payload, ":")
	if idx <= 0 || idx >= len(payload)-1 {
		return "", 0, false
	}
	username := payload[:idx]
	iatUnix, err := strconv.ParseInt(payload[idx+1:], 10, 64)
	if err != nil {
		return "", 0, false
	}

	expected := signTokenWithIAT(username, secret, iatUnix)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return "", 0, false
	}
	return username, iatUnix, true
}

// signPendingToken creates a short-lived "password verified, awaiting 2FA" token.
// The embedded principal is "2fa:<username>" so a pending token can never be
// mistaken for a full session token (validSession rejects it because no user
// named "2fa:alice" exists).
func signPendingToken(username, secret string) string {
	return signToken("2fa:"+username, secret)
}

// verifyPendingToken validates a pending-2FA token and returns the bare username.
func verifyPendingToken(token, secret string) (string, bool) {
	raw, ok := verifyToken(token, secret)
	if !ok || !strings.HasPrefix(raw, "2fa:") {
		return "", false
	}
	return strings.TrimPrefix(raw, "2fa:"), true
}
