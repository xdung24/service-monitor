package monitor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xdung24/conductor/internal/models"
)

// ---------------------------------------------------------------------------
// isStatusAccepted unit tests (no network)
// ---------------------------------------------------------------------------

func TestIsStatusAccepted_DefaultRange(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{301, true},
		{302, true},
		{399, true},
		{400, false},
		{404, false},
		{500, false},
	}
	for _, tt := range tests {
		got := isStatusAccepted(tt.code, "")
		if got != tt.want {
			t.Errorf("isStatusAccepted(%d, \"\") = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestIsStatusAccepted_CustomList(t *testing.T) {
	tests := []struct {
		code     int
		accepted string
		want     bool
	}{
		{200, "200,201,204", true},
		{201, "200,201,204", true},
		{404, "200,201,204", false},
		{404, "200,404", true},
		{500, "500", true},
		{200, " 200 , 201 ", true}, // whitespace tolerance
	}
	for _, tt := range tests {
		got := isStatusAccepted(tt.code, tt.accepted)
		if got != tt.want {
			t.Errorf("isStatusAccepted(%d, %q) = %v, want %v", tt.code, tt.accepted, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// HTTPChecker integration tests using httptest.Server
// ---------------------------------------------------------------------------

func baseMonitor(url string) *models.Monitor {
	return &models.Monitor{
		URL:              url,
		TimeoutSeconds:   5,
		HTTPMethod:       "GET",
		HTTPMaxRedirects: 10,
	}
}

func TestHTTPChecker_StatusOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := (&HTTPChecker{}).Check(context.Background(), baseMonitor(srv.URL))
	if r.Status != 1 {
		t.Errorf("want UP, got DOWN: %s", r.Message)
	}
}

func TestHTTPChecker_CustomAcceptedStatuses_404AsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPAcceptedStatuses = "200,404"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("404 with custom statuses: want UP, got DOWN: %s", r.Message)
	}
}

func TestHTTPChecker_CustomAcceptedStatuses_500Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPAcceptedStatuses = "200"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Errorf("500 should be DOWN, got UP: %s", r.Message)
	}
}

func TestHTTPChecker_KeywordFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello service monitor world")
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPKeyword = "service monitor"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("keyword found: want UP, got DOWN: %s", r.Message)
	}
}

func TestHTTPChecker_KeywordNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello world")
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPKeyword = "missing phrase"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Errorf("keyword missing: want DOWN, got UP: %s", r.Message)
	}
}

func TestHTTPChecker_KeywordInvert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello world")
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPKeyword = "missing phrase"
	m.HTTPKeywordInvert = true
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("inverted keyword (not found): want UP, got DOWN: %s", r.Message)
	}
}

func TestHTTPChecker_BasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if ok && u == "admin" && p == "secret" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	// Without credentials: 401 → DOWN (default accepted range)
	r := (&HTTPChecker{}).Check(context.Background(), baseMonitor(srv.URL))
	if r.Status != 0 {
		t.Error("no auth: want DOWN")
	}

	// With correct credentials: 200 → UP
	m := baseMonitor(srv.URL)
	m.HTTPUsername = "admin"
	m.HTTPPassword = "secret"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("basic auth: want UP, got: %s", r.Message)
	}
}

func TestHTTPChecker_BearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer tok123" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer srv.Close()

	// Without token: 403 → DOWN
	r := (&HTTPChecker{}).Check(context.Background(), baseMonitor(srv.URL))
	if r.Status != 0 {
		t.Error("no token: want DOWN")
	}

	// With token: 200 → UP
	m := baseMonitor(srv.URL)
	m.HTTPBearerToken = "tok123"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("bearer: want UP, got: %s", r.Message)
	}
}

func TestHTTPChecker_BearerTakesPriorityOverBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer mybearer" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPUsername = "user"
	m.HTTPPassword = "pass"
	m.HTTPBearerToken = "mybearer"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("bearer priority: want UP, got: %s", r.Message)
	}
}

func TestHTTPChecker_NoRedirect(t *testing.T) {
	redirected := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/target", http.StatusFound)
		} else {
			redirected = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	// MaxRedirects=0: don't follow, get 302 which is in 2xx/3xx range → UP
	m := baseMonitor(srv.URL)
	m.HTTPMaxRedirects = 0
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if redirected {
		t.Error("redirect should not have been followed")
	}
	if r.Status != 1 {
		t.Errorf("302 with no-follow: want UP (302 in accepted range), got: %s", r.Message)
	}
}

func TestHTTPChecker_HTTPMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPMethod = "HEAD"
	(&HTTPChecker{}).Check(context.Background(), m)
	if gotMethod != "HEAD" {
		t.Errorf("HTTP method: want HEAD, got %q", gotMethod)
	}
}
