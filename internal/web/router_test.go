package web

import (
	"html/template"
	"testing"
)

// TestTemplatesParse verifies that all HTML templates parse without errors at
// build / test time. Because templates are embedded via embed.FS, any syntax
// error or missing FuncMap entry will be caught here before a binary is
// produced.
func TestTemplatesParse(t *testing.T) {
	_, err := template.New("").Funcs(templateFuncMap()).ParseFS(templateFS, "templates/*.gohtml")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}
}
