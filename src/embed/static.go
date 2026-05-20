package embed

import "embed"

//go:embed static/* templates/index.html
var StaticFiles embed.FS
