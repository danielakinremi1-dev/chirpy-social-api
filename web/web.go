// Package web holds the static files served under /app/.
package web

import "embed"

//go:embed index.html assets
var FS embed.FS
