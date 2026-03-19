package monitor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// compareExpectedValue unit tests
// ---------------------------------------------------------------------------

func TestCompareExpectedValue(t *testing.T) {
	tests := []struct {
		actual   string
		expected string
		want     bool
	}{
		// Exact
		{"active", "active", true},
		{"active", "inactive", false},
		{"", "", true},
		// Contains (~)
		{"hello world", "~world", true},
		{"hello world", "~missing", false},
		// Not-equal (!=)
		{"active", "!=inactive", true},
		{"active", "!=active", false},
		// Numeric >
		{"10", ">5", true},
		{"5", ">5", false},
		{"4", ">5", false},
		// Numeric >=
		{"5", ">=5", true},
		{"6", ">=5", true},
		{"4", ">=5", false},
		// Numeric <
		{"3", "<5", true},
		{"5", "<5", false},
		// Numeric <=
		{"5", "<=5", true},
		{"4", "<=5", true},
		{"6", "<=5", false},
		// Fallback to lexicographic when non-numeric
		{"b", ">a", true},
		{"a", ">b", false},
	}
	for _, tt := range tests {
		got := compareExpectedValue(tt.actual, tt.expected)
		if got != tt.want {
			t.Errorf("compareExpectedValue(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// evalJsonPath unit tests
// ---------------------------------------------------------------------------

func TestEvalJsonPath(t *testing.T) {
	body := []byte(`{
		"status": "active",
		"count": 42,
		"ratio": 3.14,
		"flag": true,
		"nested": {"ok": true, "msg": "hello"},
		"items": [10, 20, 30],
		"objs": [{"id": 1}, {"id": 2}]
	}`)

	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"$", "", false}, // root → JSON object (non-empty)
		{"$.status", "active", false},
		{"$.count", "42", false},
		{"$.ratio", "3.14", false},
		{"$.flag", "true", false},
		{"$.nested.ok", "true", false},
		{"$.nested.msg", "hello", false},
		{"$.items[0]", "10", false},
		{"$.items[1]", "20", false},
		{"$.items[-1]", "30", false}, // negative index
		{"$.objs[0].id", "1", false},
		{"$.objs[1].id", "2", false},
		{"$.missing", "", true},   // key not found
		{"$.items[99]", "", true}, // index out of range
		{"$.count.bad", "", true}, // not an object
	}
	for _, tt := range tests {
		got, err := evalJsonPath(body, tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("evalJsonPath(%q): err=%v, wantErr=%v", tt.path, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && tt.want != "" && got != tt.want {
			t.Errorf("evalJsonPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestEvalJsonPath_ArrayRoot(t *testing.T) {
	body := []byte(`[{"name":"alice"},{"name":"bob"}]`)
	got, err := evalJsonPath(body, "$[0].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "alice" {
		t.Errorf("want alice, got %q", got)
	}
}

func TestEvalJsonPath_InvalidJSON(t *testing.T) {
	_, err := evalJsonPath([]byte(`not-json`), "$.key")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// Header check integration tests
// ---------------------------------------------------------------------------

func TestHTTPChecker_HeaderExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Health", "ok")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Presence check only — UP.
	m := baseMonitor(srv.URL)
	m.HTTPHeaderName = "X-Health"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("header presence: want UP, got: %s", r.Message)
	}

	// Correct value — UP.
	m.HTTPHeaderValue = "ok"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("header value match: want UP, got: %s", r.Message)
	}

	// Wrong value — DOWN.
	m.HTTPHeaderValue = "bad"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("header value mismatch: want DOWN, got UP")
	}
}

func TestHTTPChecker_HeaderMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPHeaderName = "X-Required-Header"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Errorf("missing header: want DOWN, got UP: %s", r.Message)
	}
}

// ---------------------------------------------------------------------------
// Body type check integration tests
// ---------------------------------------------------------------------------

func TestHTTPChecker_BodyTypeJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPBodyType = "json"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("json body type match: want UP, got: %s", r.Message)
	}

	// Expecting xml but got json — DOWN.
	m.HTTPBodyType = "xml"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("json response with xml body type: want DOWN, got UP")
	}
}

func TestHTTPChecker_BodyTypeXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<root><status>ok</status></root>`)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPBodyType = "xml"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("xml body type: want UP, got: %s", r.Message)
	}
}

func TestHTTPChecker_BodyTypeText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPBodyType = "text"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("text body type: want UP, got: %s", r.Message)
	}

	// JSON response with text body type — DOWN.
	srvJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srvJSON.Close()
	m2 := baseMonitor(srvJSON.URL)
	m2.HTTPBodyType = "text"
	r2 := (&HTTPChecker{}).Check(context.Background(), m2)
	if r2.Status != 0 {
		t.Error("json response with text body type: want DOWN")
	}
}

// ---------------------------------------------------------------------------
// JSON path integration tests
// ---------------------------------------------------------------------------

func TestHTTPChecker_JsonPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"active","count":42,"nested":{"ok":true},"items":[1,2,3]}`)
	}))
	defer srv.Close()

	tests := []struct {
		path     string
		expected string
		wantUP   bool
	}{
		{"$.status", "active", true},
		{"$.status", "inactive", false},
		{"$.count", "42", true},
		{"$.count", ">10", true},
		{"$.count", "<10", false},
		{"$.count", ">=42", true},
		{"$.count", "<=41", false},
		{"$.count", "!=99", true},
		{"$.nested.ok", "true", true},
		{"$.items[0]", "1", true},
		{"$.items[2]", "3", true},
		{"$.status", "~ctiv", true}, // contains
		{"$.missing", "", false},    // key not found → error → DOWN
	}
	for _, tt := range tests {
		m := baseMonitor(srv.URL)
		m.HTTPJsonPath = tt.path
		m.HTTPJsonExpected = tt.expected
		r := (&HTTPChecker{}).Check(context.Background(), m)
		if (r.Status == 1) != tt.wantUP {
			if tt.wantUP {
				t.Errorf("jsonpath(%q, %q): want UP, got DOWN: %s", tt.path, tt.expected, r.Message)
			} else {
				t.Errorf("jsonpath(%q, %q): want DOWN, got UP", tt.path, tt.expected)
			}
		}
	}
}

func TestHTTPChecker_JsonPath_ExistenceOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","nothing":null}`)
	}))
	defer srv.Close()

	// Path exists with non-null value — UP.
	m := baseMonitor(srv.URL)
	m.HTTPJsonPath = "$.status"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("existence check (non-null): want UP, got: %s", r.Message)
	}

	// Path resolves to null — DOWN.
	m.HTTPJsonPath = "$.nothing"
	r = (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("existence check (null value): want DOWN, got UP")
	}
}

// ---------------------------------------------------------------------------
// XPath integration tests
// ---------------------------------------------------------------------------

func TestHTTPChecker_XPath(t *testing.T) {
	xmlBody := `<?xml version="1.0"?>
<root>
  <status>active</status>
  <count>10</count>
  <nested><msg>hello</msg></nested>
</root>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, xmlBody)
	}))
	defer srv.Close()

	tests := []struct {
		expr     string
		expected string
		wantUP   bool
	}{
		{"//status", "active", true},
		{"//status", "inactive", false},
		{"//count", "10", true},
		{"//count", ">5", true},
		{"//count", "<5", false},
		{"//nested/msg", "hello", true},
		{"//notexist", "", false}, // no matching node
	}
	for _, tt := range tests {
		m := baseMonitor(srv.URL)
		m.HTTPXPath = tt.expr
		m.HTTPXPathExpected = tt.expected
		r := (&HTTPChecker{}).Check(context.Background(), m)
		if (r.Status == 1) != tt.wantUP {
			if tt.wantUP {
				t.Errorf("xpath(%q, %q): want UP, got DOWN: %s", tt.expr, tt.expected, r.Message)
			} else {
				t.Errorf("xpath(%q, %q): want DOWN, got UP", tt.expr, tt.expected)
			}
		}
	}
}

func TestHTTPChecker_XPath_SOAP(t *testing.T) {
	soapBody := `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:ns="http://example.com/service">
  <soap:Body>
    <ns:GetStatusResponse>
      <ns:status>ok</ns:status>
      <ns:code>200</ns:code>
    </ns:GetStatusResponse>
  </soap:Body>
</soap:Envelope>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, soapBody)
	}))
	defer srv.Close()

	// XPath with local-name() function to match namespace-prefixed elements.
	m := baseMonitor(srv.URL)
	m.HTTPXPath = "//*[local-name()='status']"
	m.HTTPXPathExpected = "ok"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("SOAP xpath: want UP, got: %s", r.Message)
	}
}

func TestHTTPChecker_XPath_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `not xml at all {{{{`)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.HTTPXPath = "//status"
	r := (&HTTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("invalid XML with XPath assertion: want DOWN, got UP")
	}
}
