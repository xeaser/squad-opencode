package opencodestore

import (
	"database/sql"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// OpenReadOnly opens path with a WAL-safe read-only URI.
func OpenReadOnly(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := "file:" + filepath.ToSlash(abs) + "?mode=ro"
	return sql.Open("sqlite", uri)
}
