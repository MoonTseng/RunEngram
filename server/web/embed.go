// Package web exposes the bundled React UI as an io/fs.FS to the server.
//
// Two delivery paths, in priority:
//
//  1. **Embedded** — anything inside ./dist at build time is compiled into
//     the binary via go:embed. A single placeholder file (.gitkeep) lives
//     there so the embed directive succeeds even on a fresh checkout that
//     hasn't run `pnpm build` yet; FS() detects this and falls through.
//
//  2. **External directory** — if `./dev-web/` exists relative to CWD,
//     serve from there. Useful when iterating on the React app without
//     rebuilding the server binary every time.
//
// If neither yields a real bundle, FS returns ok=false and the handler
// falls back to API-only mode.
package web

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the static asset filesystem and true when a bundle is
// available, or (nil, false) when the server should run API-only.
func FS() (fs.FS, bool) {
	if sub, err := fs.Sub(embedded, "dist"); err == nil {
		entries, _ := fs.ReadDir(sub, ".")
		// More than just the .gitkeep placeholder = real bundle.
		if len(entries) > 1 || (len(entries) == 1 && entries[0].Name() != ".gitkeep") {
			return sub, true
		}
	}
	candidate := filepath.Join(".", "dev-web")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return os.DirFS(candidate), true
	}
	return nil, false
}
