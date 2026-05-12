package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var migrationsFS embed.FS

func FS() fs.FS {
	return migrationsFS
}
