//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedFS embed.FS

// DashboardFS is the embedded dashboard, rooted at the dist/ directory.
var DashboardFS fs.FS = embeddedFS
