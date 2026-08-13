package squad

import (
	"embed"
	"io/fs"
)

// Templates holds default preset files under templates/.
//
//go:embed all:templates
var embeddedTemplates embed.FS

// TemplateFS returns the embedded template filesystem rooted at "templates".
func TemplateFS() fs.FS {
	sub, err := fs.Sub(embeddedTemplates, "templates")
	if err != nil {
		// templates must exist at build time
		panic(err)
	}
	return sub
}
