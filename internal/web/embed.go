package web

import "embed"

//go:embed templates/*.gohtml
var templateFS embed.FS
