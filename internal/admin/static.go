package admin

import (
	"embed"
	"io/fs"
)

//go:embed static/style.css static/logo.svg static/favicon.svg static/fonts/*.woff2
var staticFS embed.FS

func staticFileSystem() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
