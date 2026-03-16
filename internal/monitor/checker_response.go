package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"

	"github.com/xdung24/service-monitor/internal/models"
)

// checkHeaderConstraint verifies a response header presence/value assertion.
// Returns "" on pass, or a human-readable failure description.
func checkHeaderConstraint(resp *http.Response, m *models.Monitor) string {
	if m.HTTPHeaderName == "" {
		return ""
	}
	canonKey := http.CanonicalHeaderKey(m.HTTPHeaderName)
	vals, exists := resp.Header[canonKey]
	if !exists || len(vals) == 0 {
		return fmt.Sprintf("header %q is missing from response", m.HTTPHeaderName)
	}
	if m.HTTPHeaderValue == "" {
		return "" // presence-only check passes
	}
	for _, v := range vals {
		if v == m.HTTPHeaderValue {
			return ""
		}
	}
	return fmt.Sprintf("header %q: expected %q, got %q",
		m.HTTPHeaderName, m.HTTPHeaderValue, strings.Join(vals, ", "))
}

// checkBodyType validates the response Content-Type matches the expected type.
// bodyType values (from monitor config): "", "json", "xml", "text", "binary"
func checkBodyType(resp *http.Response, bodyType string) string {
	if bodyType == "" {
		return ""
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	switch bodyType {
	case "json":
		if !strings.Contains(ct, "json") {
			return fmt.Sprintf("body type: expected JSON (Content-Type with 'json'), got %q", ct)
		}
	case "xml":
		if !strings.Contains(ct, "xml") {
			return fmt.Sprintf("body type: expected XML/SOAP (Content-Type with 'xml'), got %q", ct)
		}
	case "text":
		if !strings.HasPrefix(ct, "text/") {
			return fmt.Sprintf("body type: expected text/* Content-Type, got %q", ct)
		}
	case "binary":
		if strings.HasPrefix(ct, "text/") || strings.Contains(ct, "json") || strings.Contains(ct, "xml") {
			return fmt.Sprintf("body type: expected binary/file Content-Type, got %q", ct)
		}
	}
	return ""
}

// checkJsonPath evaluates a JSONPath expression on body and compares to expected.
// An empty expected value only asserts the path resolves to a non-null value.
func checkJsonPath(body []byte, path, expected string) string {
	if path == "" {
		return ""
	}
	actual, err := evalJsonPath(body, path)
	if err != nil {
		return fmt.Sprintf("JSONPath %q: %v", path, err)
	}
	if expected == "" {
		if actual == "null" {
			return fmt.Sprintf("JSONPath %q resolved to null", path)
		}
		return ""
	}
	if !compareExpectedValue(actual, expected) {
		return fmt.Sprintf("JSONPath %q: expected %q, got %q", path, expected, actual)
	}
	return ""
}

// checkXPath evaluates an XPath expression on XML/SOAP body and compares to expected.
// An empty expected value only asserts that a matching node exists.
func checkXPath(body []byte, expr, expected string) string {
	if expr == "" {
		return ""
	}
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return fmt.Sprintf("XPath: cannot parse XML body: %v", err)
	}
	node := xmlquery.FindOne(doc, expr)
	if node == nil {
		return fmt.Sprintf("XPath %q: no matching node in response", expr)
	}
	if expected == "" {
		return ""
	}
	actual := strings.TrimSpace(node.InnerText())
	if !compareExpectedValue(actual, expected) {
		return fmt.Sprintf("XPath %q: expected %q, got %q", expr, expected, actual)
	}
	return ""
}

// ---------------------------------------------------------------------------
// JSONPath evaluator
// ---------------------------------------------------------------------------

// evalJsonPath evaluates a JSONPath expression and returns the matched value
// as a string. Supported syntax:
//
//	$           — root value
//	$.key       — object key
//	$.a.b.c     — nested keys
//	$.arr[0]    — array index (negative indices count from end)
//	$.arr[0].k  — array index then key
//	$[0].key    — root is array
func evalJsonPath(body []byte, path string) (string, error) {
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("invalid JSON body: %v", err)
	}
	expr := strings.TrimPrefix(path, "$")
	if expr == "" {
		return jsonValueToString(root), nil
	}
	if expr[0] != '.' && expr[0] != '[' {
		return "", fmt.Errorf("invalid JSONPath: must start with $, $. or $[")
	}
	steps, err := tokenizeJsonPath(expr)
	if err != nil {
		return "", err
	}
	current := root
	for _, step := range steps {
		current, err = applyJsonStep(current, step)
		if err != nil {
			return "", err
		}
	}
	return jsonValueToString(current), nil
}

type jsonPathStep struct {
	key   string
	index int
	isIdx bool
}

// tokenizeJsonPath splits a JSONPath expression (after the leading $) into steps.
func tokenizeJsonPath(expr string) ([]jsonPathStep, error) {
	expr = strings.TrimPrefix(expr, ".")
	var steps []jsonPathStep
	for expr != "" {
		if expr[0] == '[' {
			end := strings.IndexByte(expr, ']')
			if end == -1 {
				return nil, fmt.Errorf("JSONPath: unclosed '['")
			}
			idxStr := strings.TrimSpace(expr[1:end])
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("JSONPath: invalid array index %q", idxStr)
			}
			steps = append(steps, jsonPathStep{index: idx, isIdx: true})
			expr = expr[end+1:]
			expr = strings.TrimPrefix(expr, ".")
			continue
		}
		// Locate key boundary (next '.' or '[' or end-of-string).
		end := len(expr)
		if i := strings.IndexByte(expr, '.'); i != -1 && i < end {
			end = i
		}
		if i := strings.IndexByte(expr, '['); i != -1 && i < end {
			end = i
		}
		key := expr[:end]
		if key == "" {
			return nil, fmt.Errorf("JSONPath: empty key segment")
		}
		steps = append(steps, jsonPathStep{key: key})
		expr = expr[end:]
		expr = strings.TrimPrefix(expr, ".")
	}
	return steps, nil
}

func applyJsonStep(current interface{}, step jsonPathStep) (interface{}, error) {
	if step.isIdx {
		arr, ok := current.([]interface{})
		if !ok {
			return nil, fmt.Errorf("JSONPath: expected array for index step, got %T", current)
		}
		idx := step.index
		if idx < 0 {
			idx = len(arr) + idx
		}
		if idx < 0 || idx >= len(arr) {
			return nil, fmt.Errorf("JSONPath: index %d out of range (len %d)", step.index, len(arr))
		}
		return arr[idx], nil
	}
	obj, ok := current.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("JSONPath: expected object for key %q, got %T", step.key, current)
	}
	val, exists := obj[step.key]
	if !exists {
		return nil, fmt.Errorf("JSONPath: key %q not found", step.key)
	}
	return val, nil
}

// jsonValueToString converts any JSON-decoded value to its string representation.
func jsonValueToString(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// Render integers without a decimal point.
		if t == float64(int64(t)) && t >= -1e15 && t <= 1e15 {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// ---------------------------------------------------------------------------
// Value comparison
// ---------------------------------------------------------------------------

// compareExpectedValue checks whether actual satisfies the expected pattern.
//
// Supported prefix operators:
//
//	(none) — exact string equality
//	~      — actual contains the suffix (e.g. ~active)
//	!=     — not equal
//	>  >=  <  <=  — numeric comparison; falls back to lexicographic
func compareExpectedValue(actual, expected string) bool {
	switch {
	case strings.HasPrefix(expected, ">="):
		return valueCmp(actual, expected[2:]) >= 0
	case strings.HasPrefix(expected, "<="):
		return valueCmp(actual, expected[2:]) <= 0
	case strings.HasPrefix(expected, "!="):
		return actual != expected[2:]
	case strings.HasPrefix(expected, ">"):
		return valueCmp(actual, expected[1:]) > 0
	case strings.HasPrefix(expected, "<"):
		return valueCmp(actual, expected[1:]) < 0
	case strings.HasPrefix(expected, "~"):
		return strings.Contains(actual, expected[1:])
	default:
		return actual == expected
	}
}

// valueCmp compares two values numerically when both parse as float64,
// otherwise falls back to lexicographic string comparison.
func valueCmp(a, b string) int {
	fa, errA := strconv.ParseFloat(a, 64)
	fb, errB := strconv.ParseFloat(b, 64)
	if errA == nil && errB == nil {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		default:
			return 0
		}
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
