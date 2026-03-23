package web

import "embed"

//go:embed templates/*.gohtml
var templateFS embed.FS

//go:embed public
var publicFS embed.FS
