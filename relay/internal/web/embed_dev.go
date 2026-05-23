//go:build !embed

// Package web holds the embedded dashboard assets.
// In development (no embed tag), DashboardFS is nil and the relay
// does not serve a dashboard — use the Vite dev server instead.
package web

import "io/fs"

// DashboardFS is nil when built without the embed build tag.
var DashboardFS fs.FS
