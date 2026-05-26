package web

import "embed"

//go:embed assets/* assets/**/*
var AssetsFS embed.FS
