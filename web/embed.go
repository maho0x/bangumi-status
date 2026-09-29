// Package web embeds the built frontend (see package.json; `npm run build`
// writes dist/). Without a build the aggregator still compiles and serves the
// API, but the page itself is missing.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS is the built site rooted at dist/.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
