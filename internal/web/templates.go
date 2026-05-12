package web

import "embed"

//go:embed templates/*.tmpl templates/**/*.tmpl
var TemplatesFS embed.FS
