package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist/*
var distFS embed.FS

// Dist renvoie le système de fichiers embarqué pour le frontend React.
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	return sub
}
