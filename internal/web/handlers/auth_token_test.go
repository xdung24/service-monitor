package handlers

import "testing"

func TestSignVerifyToken_RoundTrip(t *testing.T) {
	token := signToken("alice", "supersecret")
	username, ok := verifyToken(token, "supersecret")
	if !ok {
		t.Fatal("expected valid token to verify successfully")
	}
	if username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", username)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	token := signToken("alice", "supersecret")
	_, ok := verifyToken(token, "wrongsecret")
	if ok {
		t.Fatal("expected token signed with different secret to fail verification")
	}
}

func TestVerifyToken_TamperedUsername(t *testing.T) {
	token := signToken("alice", "supersecret")
	// replace encoded username part with a different user's encoding
	tampered := signToken("admin", "supersecret")
	// use alice's sig with admin's payload (manually constructed)
	_ = tampered
	_, ok := verifyToken("dGVzdA."+token[len(token)-43:], "supersecret")
	if ok {
		t.Fatal("expected tampered token to fail verification")
	}
}

func TestVerifyToken_NoDot(t *testing.T) {
	_, ok := verifyToken("nodotintoken", "secret")
	if ok {
		t.Fatal("expected token without dot to fail")
	}
}

func TestVerifyToken_InvalidBase64(t *testing.T) {
	_, ok := verifyToken("!!!invalid!!!.sig", "secret")
	if ok {
		t.Fatal("expected token with invalid base64 to fail")
	}
}

func TestVerifyToken_Empty(t *testing.T) {
	_, ok := verifyToken("", "secret")
	if ok {
		t.Fatal("expected empty token to fail")
	}
}
